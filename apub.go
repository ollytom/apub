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

// NormContext is a URL referencing the
// normative Activity Streams 2.0 JSON-LD @context definition.
// All [Activity] variables should have their AtContext field set to this value.
// See [Activity Streams 2.0] section 2.1.
//
// [Activity Streams 2.0]: https://www.w3.org/TR/activitystreams-core/
const NormContext string = "https://www.w3.org/ns/activitystreams"

// ContentType is the MIME media type for ActivityPub.
const ContentType string = "application/activity+json"

// PublicCollection is the ActivityPub ID for the special collection indicating public access.
// Any Activity addressed to this collection is meant to be available to all users,
// authenticated or not.
// See W3C Recommendation ActivityPub Section 5.6.
const PublicCollection string = "https://www.w3.org/ns/activitystreams#Public"

var ErrNotExist = errors.New("no such activity")

// Activity represents the Activity Streams Object core type.
// See Activity Streams 2.0, section 4.1.
type Activity struct {
	AtContext    string     `json:"@context"`
	ID           string     `json:"id"`
	Type         string     `json:"type"`
	Name         string     `json:"name,omitempty"`
	Actor        string     `json:"actor,omitempty"`
	Username     string     `json:"preferredUsername,omitempty"`
	Summary      string     `json:"summary,omitempty"`
	Inbox        string     `json:"inbox,omitempty"`
	Outbox       string     `json:"outbox,omitempty"`
	To           []string   `json:"to,omitempty"`
	CC           []string   `json:"cc,omitempty"`
	Followers    string     `json:"followers,omitempty"`
	InReplyTo    string     `json:"inReplyTo,omitempty"`
	Published    *time.Time `json:"published,omitempty"`
	AttributedTo string     `json:"attributedTo,omitempty"`
	Content      string     `json:"content,omitempty"`
	MediaType    string     `json:"mediaType,omitempty"`
	Source       struct {
		Content   string `json:"content,omitempty"`
		MediaType string `json:"mediaType,omitempty"`
	} `json:"source,omitempty"`
	PublicKey *PublicKey      `json:"publicKey,omitempty"`
	Audience  string          `json:"audience,omitempty"`
	Href      string          `json:"href,omitempty"`
	Tag       []Activity      `json:"tag,omitempty"`
	// Contains a JSON-encoded Activity, or a URL as a JSON string
	// pointing to an Activity. Use Activity.Unwrap() to access
	// the enclosed, decoded value.
	Object    json.RawMessage `json:"object,omitempty"`
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

// Unwrap returns the JSON-encoded Activity, if any, enclosed in act.
// The Activity may be referenced by ID,
// in which case the activity is looked up by client or by
// apub.defaultClient if client is nil.
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

func DecodeActor(r io.Reader) (*Actor, error) {
	a, err := Decode(r)
	if err != nil {
		return nil, err
	}
	return activityToActor(a), nil
}

type Actor struct {
	AtContext string     `json:"@context"`
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	Name      string     `json:"name"`
	Username  string     `json:"preferredUsername"`
	Summary   string     `json:"summary,omitempty"`
	Inbox     string     `json:"inbox"`
	Outbox    string     `json:"outbox"`
	Followers string     `json:"followers"`
	Published *time.Time `json:"published,omitempty"`
	PublicKey PublicKey  `json:"publicKey"`
}

type PublicKey struct {
	ID           string `json:"id"`
	Owner        string `json:"owner"`
	PublicKeyPEM string `json:"publicKeyPem"`
}

// Address generates the most likely address of the Actor.
// The Actor's name (not the username) is used as the address' proper name, if present.
// Implementors should verify the address using WebFinger.
// For example, the followers address for Actor ID
// https://hachyderm.io/users/otl is:
//
//	"Oliver Lowe" <otl+followers@hachyderm.io>
func (a *Actor) Address() *mail.Address {
	if a.Username == "" && a.Name == "" {
		return &mail.Address{"", a.ID}
	}
	trimmed := strings.TrimPrefix(a.ID, "https://")
	host, _, _ := strings.Cut(trimmed, "/")
	addr := fmt.Sprintf("%s@%s", a.Username, host)
	return &mail.Address{a.Name, addr}
}

// FollowersAddress generates a non-standard address representing the Actor's followers
// using plus addressing.
// It is the Actor's address username part with a "+followers" suffix.
// The address cannot be resolved using WebFinger.
//
// For example, the followers address for Actor ID
// https://hachyderm.io/users/otl is:
//
//	"Oliver Lowe (followers)" <otl+followers@hachyderm.io>
func (a *Actor) FollowersAddress() *mail.Address {
	if a.Followers == "" {
		return &mail.Address{"", ""}
	}
	addr := a.Address()
	user, domain, found := strings.Cut(addr.Address, "@")
	if !found {
		return &mail.Address{"", ""}
	}
	addr.Address = fmt.Sprintf("%s+followers@%s", user, domain)
	if addr.Name != "" {
		addr.Name += " (followers)"
	}
	return addr
}
