package main

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"os/user"
	"path"
	"strings"
	"time"

	"olowe.co/apub"
)

type server struct {
	fsRoot    string
	apClient  *apub.Client
	relayAddr string
}

func (srv *server) handleReceived(activity *apub.Activity) {
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
			log.Printf("unwrap apub in %s: %v", activity.ID, err)
			return
		}
		srv.handleReceived(wrapped)
		return
	default:
		return
	}
	log.Printf("relaying %s %s to %s", activity.Type, activity.ID, srv.relayAddr)
	err = srv.accept(activity)
	if err != nil {
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
	defer req.Body.Close()
	var rcv apub.Activity // received
	if err := json.NewDecoder(req.Body).Decode(&rcv); err != nil {
		log.Println("decode apub message:", err)
		stat := http.StatusBadRequest
		http.Error(w, "malformed activitypub message", stat)
		return
	}
	activity := &rcv
	if rcv.Type == "Announce" {
		var err error
		activity, err = rcv.Unwrap(nil)
		if err != nil {
			err = fmt.Errorf("unwrap apub object in %s: %w", rcv.ID, err)
			log.Println(err)
			stat := http.StatusBadRequest
			http.Error(w, err.Error(), stat)
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
		srv.deliver(activity)
		return
	case "Create", "Note", "Page", "Article":
		w.WriteHeader(http.StatusAccepted)
		srv.handleReceived(activity)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (srv *server) deliver(a *apub.Activity) error {
	p, err := apub.MarshalMail(a)
	if err != nil {
		return fmt.Errorf("marshal mail message: %w", err)
	}
	now := time.Now().Unix()
	seq := 0
	max := 99
	name := fmt.Sprintf("%d.%02d", now, seq)
	name = path.Join(srv.fsRoot, "inbox", name)
	for seq <= max {
		name = fmt.Sprintf("%d.%02d", now, seq)
		name = path.Join(srv.fsRoot, "inbox", name)
		_, err := os.Stat(name)
		if err == nil {
			seq++
			continue
		} else if errors.Is(err, fs.ErrNotExist) {
			break
		}
		return fmt.Errorf("get unique mdir name: %w", err)
	}
	if seq >= max {
		return fmt.Errorf("infinite loop to get uniqe mdir name")
	}
	return os.WriteFile(name, p, 0644)
}

func (srv *server) accept(a *apub.Activity) error {
	return apub.SendMail(srv.relayAddr, nil, "nobody", []string{"otl"}, a)
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

func newClient(keyPath string, actorPath string) (*apub.Client, error) {
	b, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("load private key: %w", err)
	}
	block, _ := pem.Decode(b)
	key, _ := x509.ParsePKCS1PrivateKey(block.Bytes)

	f, err := os.Open(actorPath)
	if err != nil {
		return nil, fmt.Errorf("load actor file: %w", err)
	}
	defer f.Close()
	actor, err := apub.DecodeActor(f)
	if err != nil {
		return nil, fmt.Errorf("decode actor: %w", err)
	}

	return &apub.Client{
		Client: http.DefaultClient,
		Key:    key,
		Actor:  actor,
	}, nil
}

func serveActorFile(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		log.Printf("%s checked %s", req.Header.Get("X-Forwarded-For"), name)
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
		http.Error(w, "oops", http.StatusInternalServerError)
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
		fsRoot:    home + "/apubtest",
		relayAddr: raddr,
	}
	fsys := os.DirFS(srv.fsRoot)
	hfsys := http.FileServer(http.FS(fsys))
	http.HandleFunc("/actor.json", serveActorFile(home+"/apubtest/actor.json"))
	http.HandleFunc("/.well-known/webfinger", serveWebFingerFile)
	http.Handle("/", hfsys)
	http.HandleFunc("/outbox/", serveActivityFile(hfsys))
	http.HandleFunc("/inbox", srv.handleInbox)
	log.Fatal(http.ListenAndServe("[::1]:8082", nil))
}
