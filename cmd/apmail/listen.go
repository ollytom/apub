package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
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

	if err := apub.SendMail(srv.relayAddr, nil, "nobody", []string{username}, activity); err != nil {
		log.Printf("relay %s via SMTP: %v", activity.ID, err)
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
	username := path.Dir(req.URL.Path)
	_, err := user.Lookup(username)
	if _, ok := err.(user.UnknownUserError); ok {
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
		log.Printf("accepted %s %s for relay to %s", activity.Type, activity.ID, username)
		go srv.relay(username, activity)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

var home string = os.Getenv("HOME")

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

func serveWebFingerFile(w http.ResponseWriter, req *http.Request) {
	if !req.URL.Query().Has("resource") {
		http.Error(w, "missing resource query parameter", http.StatusBadRequest)
		return
	}
	q := req.URL.Query().Get("resource")
	if !strings.HasPrefix(q, "acct:") {
		http.Error(w, "only acct resource lookup supported", http.StatusNotImplemented)
		return
	}
	addr := strings.TrimPrefix(q, "acct:")
	username, _, ok := strings.Cut(addr, "@")
	if !ok {
		http.Error(w, "bad acct lookup: missing @ in address", http.StatusBadRequest)
		return
	}
	fname, err := apub.UserWebFingerFile(username)
	if _, ok := err.(user.UnknownUserError); ok {
		http.Error(w, "no such user", http.StatusNotFound)
		return
	} else if err != nil {
		log.Println(err)
		stat := http.StatusInternalServerError
		http.Error(w, http.StatusText(stat), stat)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	http.ServeFile(w, req, fname)
}

const usage string = "usage: apmail [address]"

func main() {
	if len(os.Args) > 2 {
		log.Fatal(usage)
	}

	current, err := user.Current()
	if err != nil {
		log.Fatalf("lookup current user: %v", err)
	}
	acceptFor := []user.User{*current}

	raddr := "[::1]:smtp"
	if len(os.Args) == 2 {
		raddr = os.Args[1]
	}
	sclient, err := smtp.Dial(raddr)
	if err != nil {
		log.Fatal(err)
	}
	if err := sclient.Noop(); err != nil {
		log.Fatalf("check connection to %s: %v", raddr, err)
	}
	sclient.Quit()
	sclient.Close()

	srv := &server{
		relayAddr: raddr,
		acceptFor: acceptFor,
	}
	http.HandleFunc("/.well-known/webfinger", serveWebFingerFile)

	for _, u := range acceptFor {
		dataDir := path.Join(u.HomeDir, "apubtest")
		root := fmt.Sprintf("/%s/", u.Username)
		inbox := path.Join(root, "inbox")
		hfsys := serveActivityFile(http.FileServer(http.Dir(dataDir)))
		http.Handle(root, http.StripPrefix(root, hfsys))
		http.HandleFunc(inbox, srv.handleInbox)
	}
	log.Fatal(http.ListenAndServe("[::1]:8082", nil))
}
