package fs

import (
	"io/fs"
	"net/http"
	"testing"
	"testing/fstest"

	"olowe.co/apub/lemmy"
)

// ds9.lemmy.ml is a test instance run by the Lemmy maintainers.
func TestFS(t *testing.T) {
	if _, err := http.Head("https://ds9.lemmy.ml"); err != nil {
		t.Skip(err)
	}
	fsys := &FS{
		Client: &lemmy.Client{
			Address: "ds9.lemmy.ml",
			Debug:   true,
		},
	}
	_, err := fsys.Open("zzztestcommunity1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = fs.ReadFile(fsys, "zzztestcommunity1/447/331")
	if err != nil {
		t.Fatal(err)
	}

	if err := fstest.TestFS(fsys, "zzztestcommunity1", "zzztestcommunity1/447/post", "zzztestcommunity1/447/331"); err != nil {
		t.Fatal(err)
	}
}
