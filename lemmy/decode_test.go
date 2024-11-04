package lemmy

import (
	"os"
	"testing"
)

func TestPost(t *testing.T) {
	f, err := os.Open("testdata/post.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	post, creator, _, err := decodePostResponse(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(post.ID)
	if creator.ID != 151025 {
		t.Errorf("check creator ID: want %d, got %d", 2, creator.ID)
	}
	if creator.String() != "otl@lemmy.sdf.org" {
		t.Errorf("creator username: want %s, got %s", "TheAnonymouseJoker@lemmy.ml", creator.String())
	}
}
