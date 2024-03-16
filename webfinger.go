package apub

import (
	"fmt"
	"net/mail"
	"strings"

	"webfinger.net/go/webfinger"
)

// Finger wraps defaultClient.Finger.
func Finger(address string) (*Actor, error) {
	return defaultClient.Finger(address)
}

// Finger is convenience method returning the corresponding Actor,
// if any, of an address resolvable by WebFinger.
// It is equivalent to doing webfinger.Lookup then LookupActor.
func (c *Client) Finger(address string) (*Actor, error) {
	jrd, err := webfinger.Lookup(address, nil)
	if err != nil {
		return nil, err
	}
	for i := range jrd.Links {
		if jrd.Links[i].Type == ContentType {
			return c.LookupActor(jrd.Links[i].Href)
		}
	}
	return nil, ErrNotExist
}

func fingerAll(alist []*mail.Address) ([]Actor, error) {
	actors := make([]Actor, len(alist))
	for i, addr := range alist {
		q := addr.Address
		if strings.Contains(addr.Address, "+followers") {
			// strip "+followers" to get the regular address that can be fingered.
			q = strings.Replace(addr.Address, "+followers", "", 1)
		}
		actor, err := Finger(q)
		if err != nil {
			return actors, fmt.Errorf("finger %s: %w", q, err)
		}
		actors[i] = *actor
	}
	return actors, nil
}
