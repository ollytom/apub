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

func MarshalMail(activity *Activity, client *Client) ([]byte, error) {
	msg, err := marshalMail(activity, client)
	if err != nil {
		return nil, err
	}
	return encodeMsg(msg), nil
}

func marshalMail(activity *Activity, client *Client) (*mail.Message, error) {
	if client == nil {
		client = &DefaultClient
	}

	msg := new(mail.Message)
	msg.Header = make(mail.Header)
	var actors []Actor
	from, err := client.LookupActor(activity.AttributedTo)
	if err != nil {
		return nil, fmt.Errorf("build From: lookup actor %s: %w", activity.AttributedTo, err)
	}
	actors = append(actors, *from)
	msg.Header["From"] = []string{from.Address().String()}

	var addrs, collections []string
	for _, id := range activity.To {
		if id == PublicCollection {
			continue
		}

		a, err := client.LookupActor(id)
		if err != nil {
			return nil, fmt.Errorf("build To: lookup actor %s: %w", id, err)
		}
		if a.Type == "Collection" || a.Type == "OrderedCollection" {
			collections = append(collections, a.ID)
		} else {
			addrs = append(addrs, a.Address().String())
			actors = append(actors, *a)
		}
	}
	for _, id := range collections {
		if i := indexFollowers(actors, id); i >= 0 {
			addrs = append(addrs, actors[i].FollowersAddress().String())
		}
	}
	msg.Header["To"] = addrs

	addrs, collections = []string{}, []string{}
	for _, id := range activity.CC {
		if id == PublicCollection {
			continue
		}

		a, err := client.LookupActor(id)
		if err != nil {
			return nil, fmt.Errorf("build CC: lookup actor %s: %w", id, err)
		}
		if a.Type == "Collection" || a.Type == "OrderedCollection" {
			collections = append(collections, a.ID)
			continue
		}
		addrs = append(addrs, a.Address().String())
		actors = append(actors, *a)
	}
	for _, id := range collections {
		if i := indexFollowers(actors, id); i >= 0 {
			addrs = append(addrs, actors[i].FollowersAddress().String())
		}
	}
	msg.Header["CC"] = addrs

	msg.Header["Date"] = []string{activity.Published.Format(time.RFC822)}
	msg.Header["Message-ID"] = []string{"<" + activity.ID + ">"}
	msg.Header["Subject"] = []string{activity.Name}
	if activity.Audience != "" {
		msg.Header["List-ID"] = []string{"<" + activity.Audience + ">"}
	}
	if activity.InReplyTo != "" {
		msg.Header["In-Reply-To"] = []string{"<" + activity.InReplyTo + ">"}
	}

	msg.Body = strings.NewReader(activity.Content)
	msg.Header["Content-Type"] = []string{"text/html; charset=utf-8"}
	if activity.Source.Content != "" && activity.Source.MediaType == "text/markdown" {
		msg.Body = strings.NewReader(activity.Source.Content)
		msg.Header["Content-Type"] = []string{"text/plain; charset=utf-8"}
	} else if activity.MediaType == "text/markdown" {
		msg.Header["Content-Type"] = []string{"text/plain; charset=utf-8"}
	}
	return msg, nil
}

func indexFollowers(actors []Actor, id string) int {
	for i := range actors {
		if actors[i].Followers == id {
			return i
		}
	}
	return -1
}

func UnmarshalMail(msg *mail.Message, client *Client) (*Activity, error) {
	if client == nil {
		client = &DefaultClient
	}
	ct := msg.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart") {
		return nil, fmt.Errorf("cannot unmarshal from multipart message")
	}
	enc := msg.Header.Get("Content-Transfer-Encoding")
	if enc == "quoted-printable" {
		return nil, fmt.Errorf("cannot decode message with transfer encoding: %s", enc)
	}

	date, err := msg.Header.Date()
	if err != nil {
		return nil, fmt.Errorf("parse message date: %w", err)
	}
	from, err := msg.Header.AddressList("From")
	if err != nil {
		return nil, fmt.Errorf("parse From: %w", err)
	}
	wfrom, err := client.Finger(from[0].Address)
	if err != nil {
		return nil, fmt.Errorf("webfinger From: %w", err)
	}

	var wto, wcc []string
	var tags []Activity
	if msg.Header.Get("To") != "" {
		to, err := msg.Header.AddressList("To")
		// ignore missing To line. Some ActivityPub servers only have the
		// PublicCollection listed, which we don't care about.
		if err != nil {
			return nil, fmt.Errorf("parse To address list: %w", err)
		}
		actors, err := client.fingerAll(to)
		if err != nil {
			return nil, fmt.Errorf("webfinger To addresses: %w", err)
		}
		wto = make([]string, len(actors))
		for i, a := range actors {
			addr := strings.Trim(to[i].Address, "<>")
			if strings.Contains(addr, "+followers") {
				wto[i] = a.Followers
				continue
			}
			tags = append(tags, Activity{Type: "Mention", Href: a.ID, Name: "@" + addr})
			wto[i] = a.ID
		}
	}
	if msg.Header.Get("CC") != "" {
		cc, err := msg.Header.AddressList("CC")
		if err != nil {
			return nil, fmt.Errorf("parse CC address list: %w", err)
		}
		actors, err := client.fingerAll(cc)
		if err != nil {
			return nil, fmt.Errorf("webfinger CC addresses: %w", err)
		}
		wcc = make([]string, len(actors))
		for i, a := range actors {
			if strings.Contains(cc[i].Address, "+followers") {
				wcc[i] = a.Followers
				continue
			}
			wcc[i] = a.ID
		}
	}

	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, msg.Body); err != nil {
		return nil, fmt.Errorf("read message body: %v", err)
	}
	content := strings.TrimSpace(strings.ReplaceAll(buf.String(), "\r", ""))

	return &Activity{
		AtContext:    NormContext,
		Type:         "Note",
		AttributedTo: wfrom.ID,
		To:           wto,
		CC:           wcc,
		MediaType:    "text/markdown",
		Name:         strings.TrimSpace(msg.Header.Get("Subject")),
		Content:      content,
		InReplyTo:    strings.Trim(msg.Header.Get("In-Reply-To"), "<>"),
		Published:    &date,
		Tag:          tags,
	}, nil
}

func SendMail(addr string, auth smtp.Auth, from string, to []string, activity *Activity) error {
	msg, err := MarshalMail(activity, nil)
	if err != nil {
		return fmt.Errorf("marshal to mail message: %w", err)
	}
	return smtp.SendMail(addr, auth, from, to, msg)
}

func encodeMsg(msg *mail.Message) []byte {
	buf := &bytes.Buffer{}
	// Lead with "From", end with "Subject" to make some mail clients happy.
	fmt.Fprintln(buf, "From:", msg.Header.Get("From"))
	for k, v := range msg.Header {
		switch k {
		case "Subject", "From":
			continue
		default:
			fmt.Fprintf(buf, "%s: %s\n", k, strings.Join(v, ", "))
		}
	}
	fmt.Fprintln(buf, "Subject:", msg.Header.Get("Subject"))
	fmt.Fprintln(buf)
	io.Copy(buf, msg.Body)
	return buf.Bytes()
}
