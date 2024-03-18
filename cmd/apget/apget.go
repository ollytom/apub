// Command apget retrieves the ActivityPub object accessible at url
// and prints a formatted representation to the standard output.
//
// Its usage is:
//
// 	apget [-j] url
//
// The flags understood are:
//
//	-j
//		Print the activity as indented JSON.
// 		The default is a RFC5322 message.
//
// # Examples
//
// Deliver a Mastodon post to a local user using apsend:
//
// 	apget https://hachyderm.io/@otl/112093503066930591 | apsend otl
package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/user"

	"olowe.co/apub"
	"olowe.co/apub/internal/sys"
)

var jflag bool

func init() {
	log.SetFlags(0)
	log.SetPrefix("apsend: ")
	flag.BoolVar(&jflag, "j", false, "format as json")
	flag.Parse()
}

const usage = "apget [-j] url"

func main() {
	if len(flag.Args()) != 1 {
		log.Fatalln("usage:", usage)
	}
	current, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	client, err := sys.ClientFor(current.Username, "apubtest2.srcbeat.com")
	if err != nil {
		log.Println("create activitypub client for %s: %v", current.Username, err)
		log.Println("requests will not be signed")
		client = &apub.DefaultClient
	}
	activity, err := client.Lookup(flag.Args()[0])
	if err != nil {
		log.Fatalf("lookup %s: %v", flag.Args()[0], err)
	}
	if jflag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "	")
		if err := enc.Encode(activity); err != nil {
			os.Exit(1)
		}
		return
	}
	msg, err := apub.MarshalMail(activity, client)
	if err != nil {
		log.Println("marshal to mail:", err)
	}
	if _, err := os.Stdout.Write(msg); err != nil {
		log.Fatal(err)
	}
}
