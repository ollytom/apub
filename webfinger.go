package apub

import (
	"fmt"
	"net/mail"
	"strings"

	"webfinger.net/go/webfinger"
)

// Finger is convenience function returning the corresponding Actor,
// if any, of an address resolvable by WebFinger.
// It is equivalent to doing webfinger.Lookup then LookupActor.
func Finger(address string) (*Actor, error) {
	jrd, err := webfinger.Lookup(address, nil)
	if err != nil {
		return nil, err
	}
	for i := range jrd.Links {
		if jrd.Links[i].Type == ContentType {
			return LookupActor(jrd.Links[i].Href)
		}
	}
	return nil, ErrNotExist
}

func fingerAll(alist []*mail.Address) ([]Actor, error) {
	actors := make([]Actor, len(alist))
	for i, addr := range alist {
		if strings.Contains(addr.Address, "+followers") {
			addr.Address = strings.Replace(addr.Address, "+followers", "", 1)
			a, err := Finger(addr.Address)
			if err != nil {
				return actors, fmt.Errorf("finger %s: %w", addr.Address, err)
			}
			actors[i] = *a
			continue
		}
		actor, err := Finger(addr.Address)
		if err != nil {
			return actors, fmt.Errorf("finger %s: %w", addr.Address, err)
		}
		actors[i] = *actor
	}
	return actors, nil
}
