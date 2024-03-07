package apub

import (
	"os"
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
