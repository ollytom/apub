package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/mail"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"
	"time"

	"olowe.co/apub"
	"olowe.co/apub/internal/sys"
)

const usage string = "apsend [-F] [-t] rcpt ..."

// Delivers the mail message to the user's Maildir.
func deliverLocal(username string, msg []byte) error {
	u, err := user.Lookup(username)
	if err != nil {
		return err
	}
	inbox := path.Join(u.HomeDir, "Maildir/new")
	fname := fmt.Sprintf("%s/%d", inbox, time.Now().Unix())
	return os.WriteFile(fname, msg, 0664)
}

func wrapCreate(activity *apub.Activity) (*apub.Activity, error) {
	b, err := json.Marshal(activity)
	if err != nil {
		return nil, err
	}
	return &apub.Activity{
		AtContext: activity.AtContext,
		ID:        activity.ID + "-create",
		Actor:     activity.AttributedTo,
		Type:      "Create",
		Published: activity.Published,
		To:        activity.To,
		CC:        activity.CC,
		Object:    b,
	}, nil
}

var jflag bool
var tflag bool
var Fflag bool

func init() {
	log.SetFlags(0)
	log.SetPrefix("apsend: ")
	flag.BoolVar(&Fflag, "F", false, "file a copy for the sender")
	flag.BoolVar(&tflag, "t", false, "read recipients from message")
	flag.BoolVar(&jflag, "t", false, "read ActivityPub JSON")
	flag.Parse()
}

const sysName string = "apubtest2.srcbeat.com"

func main() {
	if tflag {
		log.Fatal("flag -t not implemented yet")
	}
	if len(flag.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "usage:", usage)
		os.Exit(1)
	}

	var activity *apub.Activity
	var bmsg []byte
	var err error
	if jflag {
		var err error
		activity, err = apub.Decode(os.Stdin)
		if err != nil {
			log.Fatalln("decode activity:", err)
		}
	} else {
		bmsg, err = io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		msg, err := mail.ReadMessage(bytes.NewReader(bmsg))
		if err != nil {
			log.Fatal(err)
		}
		activity, err = apub.UnmarshalMail(msg)
		if err != nil {
			log.Fatalln("unmarshal activity from message:", err)
		}
	}

	var remote []string
	for _, rcpt := range flag.Args() {
		if !strings.Contains(rcpt, "@") {
			if err := deliverLocal(rcpt, bmsg); err != nil {
				log.Printf("local delivery to %s: %v", rcpt, err)
			}
			continue
		}
		remote = append(remote, rcpt)
	}

	var gotErr bool
	if len(remote) > 0 {
		if !strings.HasPrefix(activity.AttributedTo, "https://"+sysName) {
			log.Fatalln("cannot send activity from non-local actor", activity.AttributedTo)
		}

		from, err := apub.LookupActor(activity.AttributedTo)
		if err != nil {
			log.Fatalf("lookup actor %s: %v", activity.AttributedTo, err)
		}
		client, err := sys.ClientFor(from.Username, sysName)
		if err != nil {
			log.Fatalf("apub cilent for %s: %v", from.Username, err)
		}

		// overwrite auto generated ID from mail clients
		if !strings.HasPrefix(activity.ID, "https://") {
			activity.ID = from.Outbox + "/" + strconv.Itoa(int(activity.Published.Unix()))
			bmsg, err = apub.MarshalMail(activity)
			if err != nil {
				log.Fatalf("remarshal %s activity to mail: %v", activity.Type, err)
			}
		}

		// Permit this activity for the public, too;
		// let's not pretend the fediverse is not public access.
		activity.To = append(activity.To, apub.PublicCollection)
		create, err := wrapCreate(activity)
		if err != nil {
			log.Fatalf("wrap %s %s in Create activity: %v", activity.Type, activity.ID, err)
		}

		// append outbound activities to the user's outbox so others can fetch it.
		sysuser, err := user.Lookup(from.Username)
		if err != nil {
			log.Fatalf("lookup system user from %s: %v", activity.ID, err)
		}
		outbox := path.Join(sys.UserDataDir(sysuser), "outbox")
		for _, a := range []*apub.Activity{activity, create} {
			b, err := json.Marshal(a)
			if err != nil {
				log.Fatalf("encode %s: %v", activity.ID, err)
			}
			fname := path.Base(a.ID)
			fname = path.Join(outbox, fname)
			if err := os.WriteFile(fname, b, 0644); err != nil {
				log.Fatalf("write activity to outbox: %v", err)
			}
		}

		for _, rcpt := range remote {
			ra, err := apub.Finger(rcpt)
			if err != nil {
				log.Printf("webfinger %s: %v", rcpt, err)
				gotErr = true
				continue
			}
			if _, err = client.Send(ra.Inbox, create); err != nil {
				log.Printf("send %s %s to %s: %v", activity.Type, activity.ID, rcpt, err)
				gotErr = true
			}
		}
		if Fflag {
			if err := deliverLocal(from.Username, bmsg); err != nil {
				log.Printf("file copy for %s: %v", from.Username, err)
				gotErr = true
			}
		}
	}

	if gotErr {
		os.Exit(1)
	}
}
