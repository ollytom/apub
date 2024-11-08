package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/user"
	"path"
	"strings"

	"olowe.co/apub"
	"olowe.co/apub/internal/sys"
)

func serveActor(w http.ResponseWriter, req http.Request, username string) {
	actor, err := sys.Actor(username, domain)
	if err != nil {
		// for security reasons we lie here; prevents user enumeration
		log.Println("lookup actor:", err)
		http.Error(w, "no such actor", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", apub.ContentType)
	if err := json.NewEncoder(w).Encode(actor); err != nil {
		log.Printf("encode actor %s: %v", actor.Username, err)
	}
}

func serveWebFinger(w http.ResponseWriter, req *http.Request) {
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
	jrd, err := sys.JRDFor(username, domain)
	if _, ok := err.(user.UnknownUserError); ok {
		http.Error(w, "no such user", http.StatusNotFound)
		return
	} else if err != nil {
		err = fmt.Errorf("webfinger jrd for %s: %v", username, err)
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(jrd); err != nil {
		log.Printf("encode webfinger response: %v", err)
	}
}

func (srv *server) nodeInfo() (NodeInfo, error) {
	var count int
	for _, user := range srv.acceptFor {
		dents, err := os.ReadDir(path.Join(user.HomeDir, "apubtest/outbox"))
		if err != nil {
			return NodeInfo{}, fmt.Errorf("count posts in outbox: %w", err)
		}
		count += len(dents)
	}
	return NewNodeInfo("apas", "0.0.1", len(srv.acceptFor), count), nil
}

func (srv *server) serveNodeInfo(w http.ResponseWriter, req *http.Request) {
	info, err := srv.nodeInfo()
	if err != nil {
		log.Println("serve nodeinfo:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(info); err != nil {
		log.Println("encode nodeinfo:", err)
	}
}
