package apub

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
)

func readPrivKey(name string) (*rsa.PrivateKey, error) {
	b, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func TestSign(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "http://example.invalid", strings.NewReader("hello, world!"))
	if err != nil {
		t.Fatal(err)
	}
	key, err := readPrivKey("testdata/private.pem")
	if err != nil {
		t.Fatal(err)
	}
	if err := Sign(req, key, "http://from.invalid/actor"); err != nil {
		t.Fatal(err)
	}
	fmt.Println(req.Header.Get("Digest"))
	fmt.Println(req.Header.Get("Signature"))
}
