package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strings"

	"olowe.co/apub"
)

type server struct {
	acceptFor []user.User
	relayAddr string
}

func (srv *server) relay(username string, activity *apub.Activity) {
	var err error
	switch activity.Type {
	case "Note":
		// check if we need to dereference
		if activity.Content == "" {
			activity, err = apub.Lookup(activity.ID)
			if err != nil {
				log.Printf("dereference %s %s: %v", activity.Type, activity.ID, err)
				return
			}
		}
	case "Page":
		// check if we need to dereference
		if activity.Name == "" {
			activity, err = apub.Lookup(activity.ID)
			if err != nil {
				log.Printf("dereference %s %s: %v", activity.Type, activity.ID, err)
				return
			}
		}
	case "Create", "Update":
		wrapped, err := activity.Unwrap(nil)
		if err != nil {
			log.Printf("unwrap from %s: %v", activity.ID, err)
			return
		}
		srv.relay(username, wrapped)
		return
	default:
		return
	}

	cmd := exec.Command("apsend", username)
	msg, err := apub.MarshalMail(activity)
	if err != nil {
		log.Printf("marshal %s %s to mail message: %v", activity.Type, activity.ID, err)
		return
	}
	cmd.Stdin = bytes.NewReader(msg)
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Printf("execute mailer for %s: %v", activity.ID, err)
		return
	}
}

func (srv *server) handleInbox(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		stat := http.StatusMethodNotAllowed
		http.Error(w, http.StatusText(stat), stat)
		return
	}
	if req.Header.Get("Content-Type") != apub.ContentType {
		w.Header().Set("Accept", apub.ContentType)
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}
	// url is https://example.com/{username}/inbox
	username := strings.Trim(path.Dir(req.URL.Path), "/")
	_, err := user.Lookup(username)
	if _, ok := err.(user.UnknownUserError); ok {
		log.Println("handle inbox:", err)
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		log.Println("handle inbox:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var accepted bool
	for i := range srv.acceptFor {
		if srv.acceptFor[i].Username == username {
			accepted = true
		}
	}
	if !accepted {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	defer req.Body.Close()
	var rcv apub.Activity // received
	if err := json.NewDecoder(req.Body).Decode(&rcv); err != nil {
		log.Println("decode apub message:", err)
		http.Error(w, "malformed activitypub message", http.StatusBadRequest)
		return
	}
	activity := &rcv
	if rcv.Type == "Announce" {
		var err error
		activity, err = rcv.Unwrap(nil)
		if err != nil {
			err = fmt.Errorf("unwrap apub object in %s: %w", rcv.ID, err)
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	raddr := req.RemoteAddr
	if req.Header.Get("X-Forwarded-For") != "" {
		raddr = req.Header.Get("X-Forwarded-For")
	}
	if activity.Type != "Like" && activity.Type != "Dislike" {
		log.Printf("%s %s received from %s", activity.Type, activity.ID, raddr)
	}
	switch activity.Type {
	case "Accept", "Reject":
		w.WriteHeader(http.StatusAccepted)
		return
	case "Create", "Note", "Page", "Article":
		w.WriteHeader(http.StatusAccepted)
		log.Printf("accepted %s %s for %s", activity.Type, activity.ID, username)
		go srv.relay(username, activity)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func logRequest(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// skip logging from checks by load balancer
		if req.URL.Path == "/" && req.Method == http.MethodHead {
			next.ServeHTTP(w, req)
			return
		}
		addr := req.RemoteAddr
		if req.Header.Get("X-Forwarded-For") != "" {
			addr = req.Header.Get("X-Forwarded-For")
		}
		log.Printf("%s %s %s", addr, req.Method, req.URL)
		next.ServeHTTP(w, req)
	}
}

func serveActorFile(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", apub.ContentType)
		http.ServeFile(w, req, name)
	}
}

func serveActivityFile(hfsys http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", apub.ContentType)
		hfsys.ServeHTTP(w, req)
	}
}

const usage string = "apserve"

const domain = "apubtest2.srcbeat.com"

func main() {
	if len(os.Args) > 1 {
		log.Fatalln("usage:", usage)
	}

	current, err := user.Current()
	if err != nil {
		log.Fatalf("lookup current user: %v", err)
	}
	acceptFor := []user.User{*current}

	srv := &server{
		acceptFor: acceptFor,
	}
	http.HandleFunc("/.well-known/webfinger", serveWebFinger)

	for _, u := range acceptFor {
		dataDir := path.Join(u.HomeDir, "apubtest")
		root := fmt.Sprintf("/%s/", u.Username)
		hfsys := serveActivityFile(http.FileServer(http.Dir(dataDir)))
		http.Handle(root, http.StripPrefix(root, hfsys))
		inbox := path.Join(root, "inbox")
		http.HandleFunc(inbox, srv.handleInbox)
	}
	log.Fatal(http.ListenAndServe("[::1]:8082", nil))
}
