package apub

import (
	"os"
	"path"
	"reflect"
	"sort"
	"testing"
)

func TestDecode(t *testing.T) {
	f, err := os.Open("testdata/announce1.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	a, err := Decode(f)
	if err != nil {
		t.Fatal("decode activity", err)
	}
	want := "https://lemmy.sdf.org/activities/like/b5bd1577-9677-4130-8312-cd2e2fd4ea44"
	inner, err := a.Unwrap(nil)
	if err != nil {
		t.Fatal("unwrap activity:", err)
	}
	if inner.ID != want {
		t.Errorf("wanted wrapped activity id %s, got %s", want, inner.ID)
	}
}

func TestTag(t *testing.T) {
	tests := []struct {
		Name    string
		Mention string
	}{
		{"testdata/note/akkoma.json", "@otl@apubtest2.srcbeat.com"},
		{"testdata/note/lemmy.json", "@Feathercrown@lemmy.world"},
		{"testdata/note/mastodon.json", "@selfhosted@lemmy.world"},
	}
	for _, tt := range tests {
		f, err := os.Open(tt.Name)
		if err != nil {
			t.Error(err)
			continue
		}
		defer f.Close()
		a, err := Decode(f)
		if err != nil {
			t.Errorf("%s: decode: %v", tt.Name, err)
			continue
		}
		if len(a.Tag) == 0 {
			t.Errorf("%s: no tags", tt.Name)
			continue
		}
		var found bool
		for _, tag := range a.Tag {
			if tag.Name == tt.Mention {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: did not find mention %s", tt.Name, tt.Mention)
		}
	}
}

func TestInboxes(t *testing.T) {
	want := []string{
		"https://apubtest2.srcbeat.com/otl/inbox",
		"https://hachyderm.io/inbox",
		"https://lemmy.world/inbox",
		"https://social.harpia.red/inbox",
	}

	root := "testdata/actor"
	dirent, err := os.ReadDir("testdata/actor")
	if err != nil {
		t.Fatal(err)
	}
	actors := make([]Actor, len(dirent))
	for i, ent := range dirent {
		f, err := os.Open(path.Join(root, ent.Name()))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		a, err := DecodeActor(f)
		if err != nil {
			t.Fatal(err)
		}
		actors[i] = *a
	}
	got := Inboxes(actors)
	sort.Strings(got)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("unexpected inbox slice of multiple actors, want %s got %s", want, got)
	}
}
