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
	flag.BoolVar(&jflag, "j", false, "read ActivityPub JSON")
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

	current, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	client, err := sys.ClientFor(current.Username, sysName)
	if err != nil {
		log.Fatalf("apub cilent for %s: %v", current.Username, err)
	}

	var activity *apub.Activity
	var bmsg []byte
	if jflag {
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
		activity, err = apub.UnmarshalMail(msg, client)
		if err != nil {
			log.Fatalln("unmarshal activity from message:", err)
		}
	}

	var remote []string
	var gotErr bool
	for _, rcpt := range flag.Args() {
		if !strings.Contains(rcpt, "@") {
			if err := deliverLocal(rcpt, bmsg); err != nil {
				gotErr = true
				log.Printf("local delivery to %s: %v", rcpt, err)
			}
			continue
		}
		remote = append(remote, rcpt)
	}
	if len(remote) > 0 {
		if !strings.HasPrefix(activity.AttributedTo, "https://"+sysName) {
			log.Fatalln("cannot send activity from non-local actor", activity.AttributedTo)
		}
		from, err := client.LookupActor(activity.AttributedTo)
		if err != nil {
			log.Fatalf("lookup actor %s: %v", activity.AttributedTo, err)
		}
		// everything we do from here onwards is on behalf of the sender,
		// so outbound requests must be signed with the sender's key.
		client, err = sys.ClientFor(from.Username, sysName)
		if err != nil {
			log.Fatalf("activitypub client for %s: %v", from.Username, err)
		}

		// overwrite auto generated ID from mail clients
		if !strings.HasPrefix(activity.ID, "https://") {
			activity.ID = from.Outbox + "/" + strconv.Itoa(int(activity.Published.Unix()))
			bmsg, err = apub.MarshalMail(activity, client)
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
		if err := sys.AppendToOutbox(from.Username, activity, create); err != nil {
			log.Fatalf("append activities to outbox: %v", err)
		}

		for _, rcpt := range remote {
			if strings.Contains(rcpt, "+followers") {
				rcpt = strings.Replace(rcpt, "+followers", "", 1)
			}
			ra, err := client.Finger(rcpt)
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
