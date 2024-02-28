package apub

import (
	"bytes"
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

func MarshalMail(activity *Activity) ([]byte, error) {
	buf := &bytes.Buffer{}

	actor, err := LookupActor(activity.AttributedTo)
	if err != nil {
		return nil, fmt.Errorf("lookup actor %s: %w", activity.AttributedTo, err)
	}
	fmt.Fprintf(buf, "From: %s\n", actor.Address())

	if activity.CC != nil {
		buf.WriteString("To: ")
		rcpt := append(activity.To, activity.CC...)
		var addrs []string
		for _, u := range rcpt {
			if u == ToEveryone {
				continue
			}
			actor, err = LookupActor(u)
			if err != nil {
				return nil, fmt.Errorf("lookup actor %s: %w", u, err)
			}
			addrs = append(addrs, actor.Address().String())
		}
		buf.WriteString(strings.Join(addrs, ", "))
		buf.WriteString("\n")
	}

	fmt.Fprintf(buf, "Date: %s\n", activity.Published.Format(time.RFC822))
	fmt.Fprintf(buf, "Message-ID: <%s>\n", activity.ID)
	if activity.Audience != "" {
		fmt.Fprintf(buf, "List-ID: <%s>\n", activity.Audience)
	}
	if activity.InReplyTo != "" {
		fmt.Fprintf(buf, "References: <%s>\n", activity.InReplyTo)
	}

	if activity.Source.Content != "" && activity.Source.MediaType == "text/markdown" {
		fmt.Fprintln(buf, "Content-Type: text/plain; charset=utf-8")
	} else {
		fmt.Fprintln(buf, "Content-Type:", activity.MediaType)
	}
	fmt.Fprintln(buf, "Subject:", activity.Name)
	fmt.Fprintln(buf)
	if activity.Source.Content != "" && activity.Source.MediaType == "text/markdown" {
		fmt.Fprintln(buf, activity.Source.Content)
	} else {
		fmt.Fprintln(buf, activity.Content)
	}
	_, err = mail.ReadMessage(bytes.NewReader(buf.Bytes()))
	return buf.Bytes(), err
}

func SendMail(client *smtp.Client, activity *Activity, from string, to ...string) error {
	b, err := MarshalMail(activity)
	if err != nil {
		return fmt.Errorf("marshal to mail message: %w", err)
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("mail command: %w", err)
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("rcpt command: %w", err)
		}
	}
	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("data command: %w", err)
	}
	if _, err := wc.Write(b); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("close message writer: %w", err)
	}
	return nil
}
