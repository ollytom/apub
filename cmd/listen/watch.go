package main

import (
	"log"
	"time"

	"olowe.co/apub/mastodon"
)

func (srv *server) watch(mastoURL, token string) {
	for {
		stream, err := mastodon.Watch(mastoURL, token)
		if err != nil {
			log.Printf("open mastodon stream: %v", err)
			return
		}
		for stream.Next() {

		}
		if stream.Err() != nil {
			log.Printf("read mastodon stream: %v", stream.Err())
		}
		time.Sleep(5)
	}
}
