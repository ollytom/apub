package main

import (
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"path"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"olowe.co/apub"
)

type server struct {
	fsRoot    string
	db        *sql.DB
	apClient  *apub.Client
	relay     *smtp.Client
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
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		srv.relay, err = smtp.Dial(srv.relayAddr)
		if err == nil {
			log.Printf("reconnected to relay %s", srv.relayAddr)
			log.Printf("retrying activity %s", activity.ID)
			srv.handleReceived(activity)
			return
		}
		log.Printf("reconnect to relay %s: %v", srv.relayAddr, err)
	}
}

func (srv *server) handleInbox(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		stat := http.StatusMethodNotAllowed
		http.Error(w, http.StatusText(stat), stat)
		return
	}
	if req.Header.Get("Content-Type") != apub.ContentType {
		stat := http.StatusUnsupportedMediaType
		http.Error(w, http.StatusText(stat), stat)
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
	err := apub.SendMail(srv.relay, a, "nobody", "otl")
	if err != nil {
		srv.relay.Quit()
		return fmt.Errorf("relay to SMTP server: %w", err)
	}
	return nil
}

var home string = os.Getenv("HOME")

const FTS5 bool = true

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
		// w.Header().Set("Content-Type", apub.ContentType)
		http.ServeFile(w, req, name)
	}
}

func main() {
	db, err := sql.Open("sqlite3", path.Join(home, "apubtest/index.db"))
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	raddr := "[::1]:smtp"
	sclient, err := smtp.Dial(raddr)
	if err != nil {
		log.Fatal(err)
	}
	if err := sclient.Noop(); err != nil {
		log.Fatalf("check connection to %s: %v", raddr, err)
	}

	srv := &server{
		fsRoot:    home + "/apubtest",
		db:        db,
		relay:     sclient,
		relayAddr: raddr,
	}
	fsys := os.DirFS(srv.fsRoot)
	hfsys := http.FileServer(http.FS(fsys))
	http.HandleFunc("/actor.json", serveActorFile(home+"/apubtest/actor.json"))
	http.HandleFunc("/", logRequest(hfsys))
	http.HandleFunc("/inbox", srv.handleInbox)
	http.HandleFunc("/search", srv.handleSearch)
	log.Fatal(http.ListenAndServe("[::1]:8082", nil))
}
