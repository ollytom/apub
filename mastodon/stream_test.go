package mastodon

import (
	"bufio"
	"os"
	"testing"
)

func TestStream(t *testing.T) {
	f, err := os.Open("posts.txt")
	if err != nil {
		t.Fatal(err)
	}
	sc := bufio.NewScanner(f)
	stream := &Stream{
		rc: f,
		sc: sc,
	}
	for stream.Next() {
		post := stream.Post()
		t.Log(post.URL)
	}
	if stream.Err() != nil {
		t.Error(stream.Err())
	}
}
