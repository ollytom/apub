package mastodon

import (
	"strings"
	"testing"
)

func TestMail(t *testing.T) {
	r := strings.NewReader(`Date: Mon, 23 Jun 2015 11:40:36 -0400
From: Gopher <from@example.com>
To: Another Gopher <to@example.com>
Subject: Gophers at Gophercon

Message body`)
	if _, err := DecodeMail(r); err != nil {
		t.Error(err)
	}
}

func TestSearch(t *testing.T) {
	root := "https://hachyderm.io/api"
	token := "T3sIJ3WIY7HjnGODcqE4_tOzDMJGtIcaFzguN511z84"
	q := "#spam"
	posts, err := Search(root, token, q)
	if err != nil {
		t.Errorf("search query %q: %v", q, err)
	}
	t.Log(posts)
}
