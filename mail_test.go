package apub

import (
	"bytes"
	"net/mail"
	"os"
	"testing"
)

func TestMail(t *testing.T) {
	want := "<Spotlight7573@lemmy.world>"

	url := "https://lemmy.world/u/Spotlight7573"
	actor, err := LookupActor(url)
	if err != nil {
		t.Fatal(err)
	}
	if actor.Address().String() != want {
		t.Errorf("got %s, want %s", actor.Address().String(), want)
	}

	f, err := os.Open("testdata/note.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	activity, err := Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	b, err := MarshalMail(activity)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(b))
	if _, err := mail.ReadMessage(bytes.NewReader(b)); err != nil {
		t.Fatal(err)
	}
}
