package apub

import (
	"fmt"
	"net/mail"
	"os"
	"os/user"
	"path"
	"strings"

	"webfinger.net/go/webfinger"
)

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

func UserWebFingerFile(username string) (string, error) {
	u, err := user.Lookup(username)
	if err != nil {
		return "", err
	}
	if u.HomeDir == "" {
		return "", fmt.Errorf("no home directory")
	}

	paths := []string{
		path.Join(u.HomeDir, "lib/webfinger"),                 // Plan 9
		path.Join(u.HomeDir, ".config/webfinger"),             // Unix-like
		path.Join(u.HomeDir, "Application Support/webfinger"), // macOS
	}
	for i := range paths {
		if _, err := os.Stat(paths[i]); err == nil {
			return paths[i], nil
		}
	}
	return "", fmt.Errorf("no webfinger file")
}
