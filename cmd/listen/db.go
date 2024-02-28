package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"olowe.co/apub"
)

func (srv *server) index(activity *apub.Activity) error {
	var who string
	if activity.AttributedTo != "" {
		who = activity.AttributedTo
	} else if activity.Actor != "" {
		who = activity.Actor
	} else {
		return fmt.Errorf("empty actor, empty attributedTo")
	}
	q := "SELECT id FROM actor WHERE id = ?"
	row := srv.db.QueryRow(q, who)
	var id string
	err := row.Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		actor, err := apub.LookupActor(who)
		if err != nil {
			return fmt.Errorf("lookup actor %s: %w", activity.AttributedTo, err)
		}
		if err := srv.indexActor(actor); err != nil {
			return fmt.Errorf("index actor %s: %w", actor.ID, err)
		}
	} else if err != nil {
		return fmt.Errorf("query index for actor %s: %w", who, err)
	}

	q = "INSERT INTO activity(id, type, name, published, summary, content, attributedTo) VALUES(?, ?, ?, ?, ?, ?, ?)"
	_, err = srv.db.Exec(q, activity.ID, activity.Type, activity.Name, activity.Published.UnixNano(), activity.Summary, activity.Content, activity.AttributedTo)
	if err != nil {
		return err
	}

	if len(activity.To) >= 1 {
		recipients := activity.To
		recipients = append(recipients, activity.CC...)
		for _, rcpt := range recipients {
			if rcpt == apub.ToEveryone {
				continue
			}
			q = "INSERT INTO recipient_to VALUES(?, ?)"
			_, err = srv.db.Exec(q, activity.ID, rcpt)
			if err != nil {
				return fmt.Errorf("insert recipient_to: %w", err)
			}
		}
	}

	if err := insertFTS(srv.db, activity); err != nil {
		return fmt.Errorf("add to full-text search: %w", err)
	}
	return nil
}

func (srv *server) indexActor(actor *apub.Actor) error {
	q := "INSERT INTO actor(id, type, name, username, published, summary) VALUES (?, ?, ?, ?, ?, ?)"
	_, err := srv.db.Exec(q, actor.ID, actor.Type, actor.Name, actor.Username, actor.Published.UnixNano(), actor.Summary)
	return err
}

func insertFTS(db *sql.DB, activity *apub.Activity) error {
	blob, err := apub.MarshalMail(activity)
	if err != nil {
		return fmt.Errorf("marshal activity to text blob: %w", err)
	}
	msg, err := mail.ReadMessage(bytes.NewReader(blob))
	if err != nil {
		return fmt.Errorf("parse intermediate mail message: %w", err)
	}
	q := `INSERT INTO post(id, "from", "to", date, in_reply_to, body) VALUES(?, ?, ?, ?, ?, ?)`
	_, err = db.Exec(q, activity.ID, msg.Header.Get("From"), msg.Header.Get("To"), msg.Header.Get("Date"), msg.Header.Get("In-Reply-To"), blob)
	return err
}

func (srv *server) handleSearch(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		stat := http.StatusMethodNotAllowed
		http.Error(w, http.StatusText(stat), stat)
		return
	}
	query := req.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "empty search query", http.StatusBadRequest)
		return
	} else if len(query) <= 3 {
		http.Error(w, "search query too short: need at least 4 characters", http.StatusBadRequest)
	}

	found, err := srv.search(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", apub.ContentType)
	if err := json.NewEncoder(w).Encode(found); err != nil {
		log.Printf("encode search results: %v", err)
	}
}

func (srv *server) search(query string) ([]apub.Activity, error) {
	var stmt string
	if FTS5 {
		stmt = "SELECT id FROM post WHERE post MATCH ? ORDER BY rank"
	} else {
		stmt = "%" + query + "%"
	}
	q := strings.ReplaceAll(query, "@", `"@"`)
	q = strings.ReplaceAll(q, ".", `"."`)
	q = strings.ReplaceAll(q, "/", `"/"`)
	q = strings.ReplaceAll(q, ":", `":"`)
	log.Printf("search %s (escaped %s)", query, q)
	rows, err := srv.db.Query(stmt, q)
	if err != nil {
		return nil, err
	}
	apids := []string{}
	for rows.Next() {
		var s string
		err := rows.Scan(&s)
		if errors.Is(err, sql.ErrNoRows) {
			return []apub.Activity{}, nil
		} else if err != nil {
			return []apub.Activity{}, err
		}
		apids = append(apids, s)
	}
	return srv.lookupActivities(apids)
}

func (srv *server) lookupActivities(apid []string) ([]apub.Activity, error) {
	q := "SELECT id, type, published, content FROM activity WHERE id " + sqlInExpr(len(apid))
	args := make([]any, len(apid))
	for i := range args {
		args[i] = any(apid[i])
	}
	rows, err := srv.db.Query(q, args...)
	if err != nil {
		return nil, err
	}

	activities := []apub.Activity{}
	for rows.Next() {
		var a apub.Activity
		var utime int64
		err := rows.Scan(&a.ID, &a.Type, &utime, &a.Content)
		if errors.Is(err, sql.ErrNoRows) {
			return activities, nil
		} else if err != nil {
			return activities, err
		}
		t := time.Unix(0, utime)
		a.Published = &t
		activities = append(activities, a)
	}
	return activities, rows.Err()
}

// sqlInExpr returns the equivalent "IN(?, ?, ?, ...)" SQL expression for the given count.
// This is only intended for use in "SELECT * WHERE id IN (?, ?, ?...)" statements.
func sqlInExpr(count int) string {
	if count <= 0 {
		return "IN ()"
	}
	return "IN (?" + strings.Repeat(", ?", count-1) + ")"
}
