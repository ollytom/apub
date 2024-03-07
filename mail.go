package apub

import (
	"bytes"
	"fmt"
	"io"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

func MarshalMail(activity *Activity) ([]byte, error) {
	buf := &bytes.Buffer{}

	from, err := LookupActor(activity.AttributedTo)
	if err != nil {
		return nil, fmt.Errorf("lookup actor %s: %w", activity.AttributedTo, err)
	}
	fmt.Fprintf(buf, "From: %s\n", from.Address())

	var rcpt []string
	for _, u := range activity.To {
		if u == PublicCollection {
			continue
		}
		actor, err := LookupActor(u)
		if err != nil {
			return nil, fmt.Errorf("lookup actor %s: %w", u, err)
		}
		rcpt = append(rcpt, actor.Address().String())
	}
	fmt.Fprintln(buf, "To:", strings.Join(rcpt, ", "))

	var rcptcc []string
	if activity.CC != nil {
		for _, u := range activity.CC {
			if u == PublicCollection {
				continue
			} else if u == from.Followers {
				rcptcc = append(rcptcc, from.FollowersAddress().String())
				continue
			}
			actor, err := LookupActor(u)
			if err != nil {
				return nil, fmt.Errorf("lookup actor %s: %w", u, err)
			}
			rcptcc = append(rcptcc, actor.Address().String())
		}
		fmt.Fprintln(buf, "CC:", strings.Join(rcptcc, ", "))
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
	} else if activity.MediaType != "" {
		fmt.Fprintln(buf, "Content-Type:", activity.MediaType)
	} else {
		fmt.Fprintln(buf, "Content-Type:", "text/html; charset=utf-8")
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

func UnmarshalMail(msg *mail.Message) (*Activity, error) {
	date, err := msg.Header.Date()
	if err != nil {
		return nil, fmt.Errorf("parse message date: %w", err)
	}
	from, err := msg.Header.AddressList("From")
	if err != nil {
		return nil, fmt.Errorf("parse From: %w", err)
	}
	wfrom, err := Finger(from[0].Address)
	if err != nil {
		return nil, fmt.Errorf("webfinger From: %w", err)
	}

	to, err := msg.Header.AddressList("To")
	if err != nil {
		return nil, fmt.Errorf("parse To address list: %w", err)
	}
	wto, err := fingerAll(to)
	if err != nil {
		return nil, fmt.Errorf("webfinger To addresses: %w", err)
	}
	var wcc []string
	if msg.Header.Get("CC") != "" {
		cc, err := msg.Header.AddressList("CC")
		if err != nil {
			return nil, fmt.Errorf("parse CC address list: %w", err)
		}
		wcc, err = fingerAll(cc)
		if err != nil {
			return nil, fmt.Errorf("webfinger CC addresses: %w", err)
		}
	}

	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, msg.Body); err != nil {
		return nil, fmt.Errorf("read message body: %v", err)
	}

	return &Activity{
		AtContext:    AtContext,
		Type:         "Note",
		AttributedTo: wfrom.ID,
		To:           wto,
		CC:           wcc,
		MediaType:    "text/markdown",
		Name:         strings.TrimSpace(msg.Header.Get("Subject")),
		Content:      strings.TrimSpace(buf.String()),
		InReplyTo:    strings.Trim(msg.Header.Get("In-Reply-To"), "<>"),
		Published:    &date,
	}, nil
}

func SendMail(addr string, auth smtp.Auth, from string, to []string, activity *Activity) error {
	msg, err := MarshalMail(activity)
	if err != nil {
		return fmt.Errorf("marshal to mail message: %w", err)
	}
	return smtp.SendMail(addr, auth, from, to, msg)
}
