package main

import (
	"fmt"
	"log"
	"os"

	"webfinger.net/go/webfinger"
)

const usage string = "webfinger addr ..."

func init() {
	log.SetFlags(0)
	log.SetPrefix("webfinger: ")
}

func main() {
	if len(os.Args) == 1 {
		log.Fatalln("usage:", usage)
	}

	var gotErr bool
	for _, addr := range os.Args[1:] {
		jrd, err := webfinger.Lookup(addr, nil)
		if err != nil {
			gotErr = true
			log.Println(err)
			continue
		}
		for i := range jrd.Links {
			fmt.Println(jrd.Links[i].Type, jrd.Links[i].Href)
		}
	}
	if gotErr {
		os.Exit(1)
	}
}