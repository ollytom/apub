package lemmy

import "testing"

func TestZeroClient(t *testing.T) {
	client := &Client{}
	if _, _, err := client.LookupCommunity("test"); err != nil {
		t.Log(err)
	}
}
