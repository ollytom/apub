package apub

import (
	"bytes"
	"errors"
	"net/mail"
	"os"
	"reflect"
	"sort"
	"strings"
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
			"<@>", // zero mail.Address
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
	tests := []struct {
		name       string
		recipients []string
	}{
		{
			"testdata/note/akkoma.json",
			[]string{
				"kariboka+followers@social.harpia.red",
				"otl@apubtest2.srcbeat.com",
			},
		},
		{
			"testdata/note/lemmy.json",
			[]string{
				"Feathercrown@lemmy.world",
				"technology@lemmy.world",
			},
		},
		{
			"testdata/note/mastodon.json",
			[]string{
				"otl+followers@hachyderm.io",
				"selfhosted+followers@lemmy.world",
				"selfhosted@lemmy.world",
			},
		},
		{
			"testdata/page.json",
			[]string{
				"technology@lemmy.world",
			},
		},
	}
	for _, tt := range tests {
		f, err := os.Open(tt.name)
		if err != nil {
			t.Error(err)
			continue
		}
		defer f.Close()
		a, err := Decode(f)
		if err != nil {
			t.Errorf("%s: decode activity: %v", tt.name, err)
			continue
		}
		b, err := MarshalMail(a, nil)
		if err != nil {
			t.Errorf("%s: marshal to mail message: %v", tt.name, err)
			continue
		}
		msg, err := mail.ReadMessage(bytes.NewReader(b))
		if err != nil {
			t.Errorf("%s: read back message from marshalled activity: %v", tt.name, err)
			continue
		}
		rcptto, err := msg.Header.AddressList("To")
		if errors.Is(err, mail.ErrHeaderNotPresent) {
			// whatever; sometimes the Activity has an empty slice.
		} else if err != nil {
			t.Errorf("%s: parse To addresses: %v", tt.name, err)
			t.Log("raw value:", msg.Header.Get("To"))
			continue
		}
		rcptcc, err := msg.Header.AddressList("CC")
		if errors.Is(err, mail.ErrHeaderNotPresent) {
			// whatever; sometimes the Activity has an empty slice.
		} else if err != nil {
			t.Errorf("%s: parse CC addresses: %v", tt.name, err)
			t.Log("raw value:", msg.Header.Get("CC"))
			continue
		}
		t.Log(rcptto)
		t.Log(rcptcc)
		rcpts := make([]string, len(rcptto)+len(rcptcc))
		for i, rcpt := range append(rcptto, rcptcc...) {
			rcpts[i] = rcpt.Address
		}
		sort.Strings(rcpts)
		if !reflect.DeepEqual(rcpts, tt.recipients) {
			t.Errorf("%s: unexpected recipients, want %s got %s", tt.name, tt.recipients, rcpts)
		}

		p := make([]byte, 8)
		if _, err := msg.Body.Read(p); err != nil {
			// Pages have no content, so skip this case
			if a.Type == "Page" {
				continue
			}
			t.Errorf("%s: read message body: %v", tt.name, err)
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
	a, err := UnmarshalMail(msg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Tag) != 1 {
		t.Fatalf("wanted 1 tag in unmarshalled activity, got %d", len(a.Tag))
	}
	want := "@henfredemars@infosec.pub"
	if a.Tag[0].Name != want {
		t.Errorf("wanted tag name %s, got %s", want, a.Tag[0].Name)
	}
	if a.MediaType != "text/markdown" {
		t.Errorf("wrong media type: wanted %s, got %s", "text/markdown", a.MediaType)
	}
	wantCC := []string{
		"https://programming.dev/c/programming",
		"https://programming.dev/u/starman",
		"https://hachyderm.io/users/otl/followers",
	}
	if !reflect.DeepEqual(wantCC, a.CC) {
		t.Errorf("wanted recipients %s, got %s", wantCC, a.CC)
	}
	if strings.Contains(a.Content, "\r") {
		t.Errorf("activity content contains carriage returns")
	}
	t.Log(a)
}
