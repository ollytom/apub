package apub

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"time"
)

// Sign signs the given HTTP request with the matching private key of the
// public key available at pubkeyURL.
func Sign(req *http.Request, key *rsa.PrivateKey, pubkeyURL string) error {
	date := time.Now().UTC().Format(http.TimeFormat)
	hash := sha256.New()
	fmt.Fprintf(hash, "(request-target): post %s\n", req.URL.Path)
	fmt.Fprintf(hash, "host: %s\n", req.URL.Host)
	fmt.Fprintf(hash, "date: %s", date)
	if req.Method == http.MethodPost {
		buf := &bytes.Buffer{}
		io.Copy(buf, req.Body)
		req.Body.Close()
		req.Body = io.NopCloser(buf)
		digest := sha256.Sum256(buf.Bytes())
		d := fmt.Sprintf("sha-256=%s", base64.StdEncoding.EncodeToString(digest[:]))
		// append a newline to the "date" key as we assumed we didn't
		// need one before.
		fmt.Fprintf(hash, "\n")
		fmt.Fprintf(hash, "digest: %s", d)
		req.Header.Set("Digest", d)
	}
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash.Sum(nil))
	if err != nil {
		return err
	}

	sigKeys := "(request-target) host date"
	if req.Method == http.MethodPost {
		sigKeys += " digest"
	}
	val := fmt.Sprintf("keyId=%q,headers=%q,signature=%q", pubkeyURL, sigKeys, base64.StdEncoding.EncodeToString(sig))

	req.Header.Set("Signature", val)
	req.Header.Set("Date", date)
	return nil
}

func Post(inbox string, key *rsa.PrivateKey, activity *Activity) error {
	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(activity); err != nil {
		return fmt.Errorf("encode activity: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, inbox, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", ContentType)
	/*
		if err := Sign(req, key, activity.Object.AttributedTo+"#main-key"); err != nil {
			return fmt.Errorf("sign request: %w", err)
		}
	*/

	b, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err = httputil.DumpResponse(resp, true)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non-ok response status %s", resp.Status)
	}
	return nil
}
