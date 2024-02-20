// apub is an implementation of the ActivityPub protocol.
//
// https://www.w3.org/TR/activitypub/
// https://www.w3.org/TR/activitystreams-core/
// https://www.w3.org/TR/activitystreams-vocabulary/
package apub

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"strings"
	"time"
)

// @context
const AtContext string = "https://www.w3.org/ns/activitystreams"

const ContentType string = "application/activity+json"

const AcceptMediaType string = `application/activity+json; profile="https://www.w3.org/ns/activitystreams"`

const ToEveryone string = "https://www.w3.org/ns/activitystreams#Public"

type Activity struct {
	AtContext    string     `json:"@context"`
	ID           string     `json:"id"`
	Type         string     `json:"type"`
	Name         string     `json:"name,omitempty"`
	Actor        string     `json:"actor,omitempty"`
	Username     string     `json:"preferredUsername,omitempty"`
	Inbox        string     `json:"inbox,omitempty"`
	Outbox       string     `json:"outbox,omitempty"`
	To           []string   `json:"to,omitempty"`
	CC           []string   `json:"cc,omitempty"`
	InReplyTo    string     `json:"inReplyTo,omitempty"`
	Published    *time.Time `json:"published,omitempty"`
	AttributedTo string     `json:"attributedTo,omitempty"`
	Content      string     `json:"content,omitempty"`
	MediaType    string     `json:"mediaType,omitempty"`
	Source       struct {
		Content   string `json:"content,omitempty"`
		MediaType string `json:"mediaType,omitempty"`
	} `json:"source,omitempty"`
	Audience string          `json:"audience,omitempty"`
	Object   json.RawMessage `json:"object,omitempty"`
}

func (act *Activity) UnmarshalJSON(b []byte) error {
	type Alias Activity
	aux := &struct {
		AtContext interface{} `json:"@context"`
		Object    interface{}
		*Alias
	}{
		Alias: (*Alias)(act),
	}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	switch v := aux.AtContext.(type) {
	case string:
		act.AtContext = v
	case []interface{}:
		if vv, ok := v[0].(string); ok {
			act.AtContext = vv
		}
	}
	return nil
}

func (act *Activity) Unwrap(client *Client) (*Activity, error) {
	if act.Object == nil {
		return nil, errors.New("no wrapped activity")
	}

	var buf io.Reader
	buf = bytes.NewReader(act.Object)
	if strings.HasPrefix(string(act.Object), "https") {
		if client == nil {
			return Lookup(string(act.Object))
		}
		return client.Lookup(string(act.Object))
	}
	return Decode(buf)
}

func Decode(r io.Reader) (*Activity, error) {
	var a Activity
	if err := json.NewDecoder(r).Decode(&a); err != nil {
		return nil, fmt.Errorf("decode activity: %w", err)
	}
	return &a, nil
}

type Actor struct {
	AtContext string    `json:"@context"`
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	Username  string    `json:"preferredUsername"`
	Inbox     string    `json:"inbox"`
	Outbox    string    `json:"outbox"`
	PublicKey PublicKey `json:"publicKey"`
}

type PublicKey struct {
	ID           string `json:"id"`
	Owner        string `json:"owner"`
	PublicKeyPEM string `json:"publicKeyPem"`
}

func (a *Actor) Address() *mail.Address {
	trimmed := strings.TrimPrefix(a.ID, "https://")
	host, _, _ := strings.Cut(trimmed, "/")
	addr := fmt.Sprintf("%s@%s", a.Username, host)
	return &mail.Address{a.Name, addr}
}
