package apub

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const requiredSigHeaders = "(request-target) host date digest"

// Sign signs the given HTTP request with the matching private key of the
// public key available at pubkeyURL.
func Sign(req *http.Request, key *rsa.PrivateKey, pubkeyURL string) error {
	if pubkeyURL == "" {
		return fmt.Errorf("no pubkey url")
	}
	date := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("Date", date)
	hash := sha256.New()
	fmt.Fprintln(hash, "(request-target):", strings.ToLower(req.Method), req.URL.Path)
	fmt.Fprintln(hash, "host:", req.URL.Hostname())
	fmt.Fprintln(hash, "date:", date)

	buf := &bytes.Buffer{}
	io.Copy(buf, req.Body)
	req.Body.Close()
	req.Body = io.NopCloser(buf)
	digest := sha256.Sum256(buf.Bytes())
	d := "SHA-256=" + base64.StdEncoding.EncodeToString(digest[:])
	fmt.Fprintf(hash, "digest: %s", d)
	req.Header.Set("Digest", d)

	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash.Sum(nil))
	if err != nil {
		return err
	}
	bsig := base64.StdEncoding.EncodeToString(sig)

	sigKeys := "(request-target) host date digest"
	val := fmt.Sprintf("keyId=%q,algorithm=%q,headers=%q,signature=%q", pubkeyURL, "rsa-sha256", sigKeys, bsig)
	req.Header.Set("Signature", val)
	return nil
}

type signature struct {
	keyID     string
	algorithm string
	headers   string
	signature string
}

func parseSignatureHeader(line string) (signature, error) {
	var sig signature
	for _, v := range strings.Split(line, ",") {
		name, val, ok := strings.Cut(v, "=")
		if !ok {
			return sig, fmt.Errorf("bad field: %s from %s", v, line)
		}
		val = strings.Trim(val, `"`)
		switch name {
		case "keyId":
			sig.keyID = val
		case "algorithm":
			sig.algorithm = val
		case "headers":
			sig.headers = val
		case "signature":
			sig.signature = val
		default:
			return signature{}, fmt.Errorf("bad field name %s", name)
		}
	}

	if sig.keyID == "" {
		return sig, fmt.Errorf("missing signature field keyId")
	} else if sig.algorithm == "" {
		return sig, fmt.Errorf("missing signature field algorithm")
	} else if sig.headers == "" {
		return sig, fmt.Errorf("missing signature field headers")
	} else if sig.signature == "" {
		return sig, fmt.Errorf("missing signature field signature")
	}
	return sig, nil
}
