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

	"olowe.co/apub"
)

var jflag bool

func init() {
	flag.BoolVar(&jflag, "j", false, "format as json")
	flag.Parse()
}

const usage = "apget [-j] url"

func main() {
	if len(flag.Args()) != 1 {
		log.Fatalln("usage:", usage)
	}
	activity, err := apub.Lookup(flag.Args()[0])
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
	msg, err := apub.MarshalMail(activity)
	if err != nil {
		log.Println(err)
	}
	if _, err := os.Stdout.Write(msg); err != nil {
		log.Fatal(err)
	}
}
