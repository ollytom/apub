package main

import (
	"bytes"
	"fmt"
	"io"
	"net/mail"
	"strings"
)

func encodMsg(msg *mail.Message) []byte {
	buf := &bytes.Buffer{}
	fmt.Fprintln(buf, msg.Header.Get("From"))
	delete(msg.Header, "From")
	for k, v := range msg.Header {
		if k == "Subject" {
			continue
		}
		fmt.Fprintf(buf, "%s: %s\n", k, strings.Join(v, ", "))
	}
	fmt.Fprintln(buf, "Subject:", msg.Header.Get("Subject"))
	fmt.Fprintln(buf)
	io.Copy(buf, msg.Body)
	return buf.Bytes()
}
