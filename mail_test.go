package apub

import (
	"bytes"
	"net/mail"
	"os"
	"testing"
)

func TestMailAddress(t *testing.T) {
	tests := []struct {
		name      string
		from      string
		followers string
	}{
		{
			"testdata/actor/mastodon.json",
			`"Oliver Lowe" <otl@hachyderm.io>`,
			`"Oliver Lowe (followers)" <otl+followers@hachyderm.io>`,
		},
		{
			"testdata/actor/akkoma.json",
			`"Kari'boka" <kariboka@social.harpia.red>`,
			`"Kari'boka (followers)" <kariboka+followers@social.harpia.red>`,
		},
		{
			"testdata/actor/lemmy.json",
			"<Spotlight7573@lemmy.world>",
			"<@>", // empty mail.Address
		},
	}
	for _, tt := range tests {
		f, err := os.Open(tt.name)
		if err != nil {
			t.Error(err)
			continue
		}
		defer f.Close()
		actor, err := DecodeActor(f)
		if err != nil {
			t.Errorf("%s: decode actor: %v", tt.name, err)
			continue
		}
		if actor.Address().String() != tt.from {
			t.Errorf("%s: from address: want %s, got %s", tt.name, tt.from, actor.Address().String())
		}
		got := actor.FollowersAddress().String()
		if got != tt.followers {
			t.Errorf("%s: followers address: want %s, got %s", tt.name, tt.followers, got)
		}
	}
}

func TestMarshalMail(t *testing.T) {
	var notes []string = []string{
		"testdata/note/akkoma.json",
		"testdata/note/lemmy.json",
		"testdata/note/mastodon.json",
		"testdata/page.json",
	}
	for _, name := range notes {
		f, err := os.Open(name)
		if err != nil {
			t.Error(err)
			continue
		}
		defer f.Close()
		a, err := Decode(f)
		if err != nil {
			t.Errorf("%s: decode activity: %v", name, err)
			continue
		}
		b, err := MarshalMail(a)
		if err != nil {
			t.Errorf("%s: marshal to mail message: %v", name, err)
			continue
		}
		if _, err := mail.ReadMessage(bytes.NewReader(b)); err != nil {
			t.Errorf("%s: read back message from marshalled activity: %v", name, err)
		}
	}
}

func TestUnmarshalMail(t *testing.T) {
	f, err := os.Open("testdata/outbound.eml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	msg, err := mail.ReadMessage(f)
	if err != nil {
		t.Fatal(err)
	}
	if testing.Short() {
		t.Skip("skipping network calls to unmarshal mail message to Activity")
	}
	if _, err := UnmarshalMail(msg); err != nil {
		t.Fatal(err)
	}
}
