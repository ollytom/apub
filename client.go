package apub

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

var defaultClient Client = Client{Client: http.DefaultClient}

func Lookup(id string) (*Activity, error) {
	return defaultClient.Lookup(id)
}

func LookupActor(id string) (*Actor, error) {
	return defaultClient.LookupActor(id)
}

type Client struct {
	*http.Client
	Key   *rsa.PrivateKey
	Actor *Actor
}

func (c *Client) Lookup(id string) (*Activity, error) {
	if !strings.HasPrefix(id, "http") {
		return nil, fmt.Errorf("id is not a HTTP URL")
	}
	if c.Client == nil {
		c.Client = http.DefaultClient
	}

	req, err := http.NewRequest(http.MethodGet, id, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", ContentType)
	if c.Key != nil && c.Actor != nil {
		if err := Sign(req, c.Key, c.Actor.PublicKey.ID); err != nil {
			return nil, fmt.Errorf("sign http request: %w", err)
		}
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
	return activityToActor(activity), nil
}

func activityToActor(activity *Activity) *Actor {
	return &Actor{
		AtContext: activity.AtContext,
		ID:        activity.ID,
		Type:      activity.Type,
		Name:      activity.Name,
		Username:  activity.Username,
		Inbox:     activity.Inbox,
		Outbox:    activity.Outbox,
		Published: activity.Published,
		Summary:   activity.Summary,
	}
}

func (c *Client) Send(inbox string, activity *Activity) (*Activity, error) {
	b, err := json.Marshal(activity)
	if err != nil {
		return nil, fmt.Errorf("encode outgoing activity: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, inbox, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", ContentType)
	if err := Sign(req, c.Key, c.Actor.PublicKey.ID); err != nil {
		return nil, fmt.Errorf("sign outgoing request: %w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case http.StatusOK:
		if resp.ContentLength == 0 {
			return nil, nil
		}
		defer resp.Body.Close()
		activity, err := Decode(resp.Body)
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		return activity, err
	case http.StatusAccepted, http.StatusNoContent:
		return nil, nil
	case http.StatusNotFound:
		return nil, fmt.Errorf("no such inbox %s", inbox)
	default:
		io.Copy(os.Stderr, resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("non-ok response status %s", resp.Status)
	}
}
