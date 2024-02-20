package apub

import (
	"os"
	"testing"
)

func TestDecode(t *testing.T) {
	samples := []string{"testdata/announce1.json", "testdata/note.json"}
	for _, name := range samples {
		f, err := os.Open(name)
		if err != nil {
			t.Error(err)
			continue
		}
		defer f.Close()
		a, err := Decode(f)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%+v", a)
	}
}
