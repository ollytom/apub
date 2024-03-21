package apub

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

var DefaultClient Client = Client{Client: http.DefaultClient}

func Lookup(id string) (*Activity, error) {
	return DefaultClient.Lookup(id)
}

func LookupActor(id string) (*Actor, error) {
	return DefaultClient.LookupActor(id)
}

type Client struct {
	*http.Client
	// Key is a RSA private key which will be used to sign requests.
	Key *rsa.PrivateKey
	// PubKeyID is a URL where the corresponding public key of Key
	// may be accessed. This must be set if Key is also set.
	PubKeyID string // actor.PublicKey.ID
}

func (c *Client) Lookup(id string) (*Activity, error) {
	if !strings.HasPrefix(id, "http") {
		return nil, fmt.Errorf("id is not a HTTP URL")
	}
	if c.Client == nil {
		c.Client = http.DefaultClient
	}

	req, err := newRequest(http.MethodGet, id, nil, c.Key, c.PubKeyID)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotExist
	} else if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("non-ok response status %s", resp.Status)
	}
	return Decode(resp.Body)
}

func (c *Client) LookupActor(id string) (*Actor, error) {
	activity, err := c.Lookup(id)
	if err != nil {
		return nil, err
	}
	switch activity.Type {
	case "Application", "Group", "Organization", "Person", "Service":
		return activityToActor(activity), nil
	case "Collection", "OrderedCollection":
		// probably followers. let caller work out what it wants to do
		return activityToActor(activity), nil
	}
	return nil, fmt.Errorf("bad object Type %s", activity.Type)
}

func activityToActor(activity *Activity) *Actor {
	actor := &Actor{
		AtContext: activity.AtContext,
		ID:        activity.ID,
		Type:      activity.Type,
		Name:      activity.Name,
		Username:  activity.Username,
		Inbox:     activity.Inbox,
		Outbox:    activity.Outbox,
		Followers: activity.Followers,
		Published: activity.Published,
		Summary:   activity.Summary,
		Endpoints: activity.Endpoints,
	}
	if activity.PublicKey != nil {
		actor.PublicKey = *activity.PublicKey
	}
	return actor
}

func (c *Client) Send(inbox string, activity *Activity) (*Activity, error) {
	b, err := json.Marshal(activity)
	if err != nil {
		return nil, fmt.Errorf("encode outgoing activity: %w", err)
	}
	req, err := newRequest(http.MethodPost, inbox, bytes.NewReader(b), c.Key, c.PubKeyID)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case http.StatusOK, http.StatusAccepted, http.StatusNoContent:
		return nil, nil
	case http.StatusNotFound:
		return nil, fmt.Errorf("no such inbox %s", inbox)
	default:
		io.Copy(os.Stderr, resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("non-ok response status %s", resp.Status)
	}
}

func newRequest(method, url string, body io.Reader, key *rsa.PrivateKey, pubkeyURL string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", ContentType)
	if body != nil {
		req.Header.Set("Content-Type", ContentType)
	}
	if key != nil {
		if err := Sign(req, key, pubkeyURL); err != nil {
			return nil, fmt.Errorf("sign request: %w", err)
		}
	}
	return req, nil
}
