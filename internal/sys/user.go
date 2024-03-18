package sys

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path"

	"olowe.co/apub"
	"webfinger.net/go/webfinger"
)

func UserDataDir(u *user.User) string {
	return path.Join(u.HomeDir, "apubtest")
}

func ConfigDir(u *user.User) (string, error) {
	paths := []string{
		path.Join(u.HomeDir, ".config/apubtest"),             // Unix-like
		path.Join(u.HomeDir, "Application Support/apubtest"), // macOS
		path.Join(u.HomeDir, "lib/apubtest"),                 // Plan 9
	}
	for i := range paths {
		if _, err := os.Stat(paths[i]); err == nil {
			return paths[i], nil
		}
	}
	return "", fmt.Errorf("no apubtest dir")
}

func Actor(name, host string) (*apub.Actor, error) {
	u, err := user.Lookup(name)
	if err != nil {
		return nil, err
	}
	uri, err := url.Parse("https://" + host)
	if err != nil {
		return nil, fmt.Errorf("bad host: %w", err)
	}
	uri.Path = path.Join("/", u.Username)
	root := uri.String()

	cdir, err := ConfigDir(u)
	if err != nil {
		return nil, fmt.Errorf("find config directory: %w", err)
	}
	pubkey, err := os.ReadFile(path.Join(cdir, "public.pem"))
	if err != nil {
		return nil, fmt.Errorf("read public key file: %w", err)
	}
	return &apub.Actor{
		AtContext: apub.NormContext,
		ID:        root + "/actor.json",
		Type:      "Person",
		Name:      u.Name,
		Username:  u.Username,
		Inbox:     root + "/inbox",
		Outbox:    root + "/outbox",
		PublicKey: apub.PublicKey{
			ID:           root + "/actor.json#main-key",
			Owner:        root + "/actor.json",
			PublicKeyPEM: string(pubkey),
		},
	}, nil
}

func ClientFor(username, host string) (*apub.Client, error) {
	sysuser, err := user.Lookup(username)
	if err != nil {
		return nil, err
	}
	actor, err := Actor(sysuser.Username, host)
	if err != nil {
		return nil, fmt.Errorf("load system actor: %w", err)
	}
	cdir, err := ConfigDir(sysuser)
	if err != nil {
		return nil, fmt.Errorf("find config dir: %w", err)
	}

	key, err := loadKey(path.Join(cdir, "private.pem"))
	if err != nil {
		return nil, fmt.Errorf("load private key: %w", err)
	}
	return &apub.Client{
		Client:   http.DefaultClient,
		Key:      key,
		PubKeyID: actor.PublicKey.ID,
	}, nil
}

func loadKey(name string) (*rsa.PrivateKey, error) {
	b, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func JRDFor(username, domain string) (*webfinger.JRD, error) {
	if _, err := user.Lookup(username); err != nil {
		return nil, err
	}
	return &webfinger.JRD{
		Subject: fmt.Sprintf("acct:%s@%s", username, domain),
		Links: []webfinger.Link{
			webfinger.Link{
				Rel:  "self",
				Type: apub.ContentType,
				Href: fmt.Sprintf("https://%s/%s/actor.json", domain, username),
			},
		},
	}, nil
}

func AppendToOutbox(username string, activities ...*apub.Activity) error {
	u, err := user.Lookup(username)
	if err != nil {
		return fmt.Errorf("lookup user: %w", err)
	}
	outbox := path.Join(UserDataDir(u), "outbox")
	for _, a := range activities {
		fname := path.Base(a.ID)
		fname = path.Join(outbox, fname)
		f, err := os.Create(fname)
		if err != nil {
			return fmt.Errorf("create file for %s: %w", a.ID, err)
		}
		if err := json.NewEncoder(f).Encode(a); err != nil {
			return fmt.Errorf("encode %s: %w", a.ID, err)
		}
		f.Close()
	}
	return nil
}
