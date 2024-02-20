package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"olowe.co/apub"
)

type server struct {
	fsRoot string
}

func (srv *server) handleReceived(activity *apub.Activity) {
	var err error
	switch activity.Type {
	default:
		return
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
			log.Printf("unwrap apub in %s: %v", wrapped.ID, err)
			return
		}
		srv.handleReceived(wrapped)
		return
	}
	if err := srv.deliver(activity); err != nil {
		log.Printf("deliver %s %s: %v", activity.Type, activity.ID, err)
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
			stat := http.StatusBadRequest
			http.Error(w, err.Error(), stat)
			return
		}
	}
	switch activity.Type {
	case "Like", "Dislike", "Delete", "Accept", "Reject":
		w.WriteHeader(http.StatusAccepted)
		return
	case "Create", "Update", "Note", "Page", "Article":
		w.WriteHeader(http.StatusAccepted)
		srv.handleReceived(activity)
		return
	}
	w.WriteHeader(http.StatusNotImplemented)
}

func (srv *server) deliver(a *apub.Activity) error {
	name := fmt.Sprintf("%d.json", time.Now().UnixNano())
	name = path.Join(srv.fsRoot, "inbox", name)
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(a); err != nil {
		return err
	}
	return nil
}

	/*
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
	*/

var home string = os.Getenv("HOME")

func main() {
	srv := &server{fsRoot: home+"/apubtest"}
	fsys := os.DirFS(srv.fsRoot)
	http.Handle("/", http.FileServer(http.FS(fsys)))
	http.HandleFunc("/inbox", srv.handleInbox)
	log.Fatal(http.ListenAndServe("[::1]:8082", nil))
}
