package main

import (
	"log"

	"github.com/emersion/go-smtp"
)

func main() {
	srv := smtp.NewServer(&Backend{})
	srv.Addr = ":2525"
	srv.Domain = "apubtest2.srcbeat.com"
	srv.AllowInsecureAuth = true

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
