// Command apget retrieves the ActivityPub object accessible at url
// and prints a formatted representation to the standard output.
//
// Its usage is:
//
// 	apget [-m] url
//
// The flags understood are:
//
//	-m
//		Print the activity as a RFC5322 message.
// 		The default is indented JSON.
//
// # Examples
//
// Deliver a Mastodon post to a local user using apsend:
//
// 	apget -m https://hachyderm.io/@otl/112093503066930591 | apsend otl
package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"olowe.co/apub"
)

var mflag bool

func init() {
	flag.BoolVar(&mflag, "m", false, "format as mail")
	flag.Parse()
}

const usage = "apget [-m] url"

func main() {
	if len(flag.Args()) != 1 {
		log.Fatalln("usage:", usage)
	}
	activity, err := apub.Lookup(flag.Args()[0])
	if err != nil {
		log.Fatalf("lookup %s: %v", flag.Args()[0], err)
	}
	if mflag {
		msg, err := apub.MarshalMail(activity)
		if err != nil {
			log.Println(err)
		}
		if _, err := os.Stdout.Write(msg); err != nil {
			log.Fatal(err)
		}
		return
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "	")
	if err := enc.Encode(activity); err != nil {
		os.Exit(1)
	}
}
