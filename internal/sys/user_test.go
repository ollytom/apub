package sys

import "testing"

func TestJRD(t *testing.T) {
	jrd, err := JRDFor("nobody", "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if jrd.Subject != "acct:nobody@example.com" {
		t.Errorf("unexpected subject %s", jrd.Subject)
	}
	if jrd.Links[0].Href != "https://example.com/nobody/actor.json" {
		t.Errorf("unexpected href %s", jrd.Links[0].Href)
	}
}
