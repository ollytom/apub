package mastodon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"
)

type Post struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
	InReplyTo string    `json:"in_reply_to_id"`
	Content   string    `json:"content"`
	URL       string    `json:"url"`
}

// DecodeMail decodes the RFC822 message-encoded post from r.
func DecodeMail(r io.Reader) (*Post, error) {
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return nil, err
	}
	var post Post
	post.InReplyTo = msg.Header.Get("In-Reply-To")
	buf := &bytes.Buffer{}
	if msg.Header.Get("Subject") != "" {
		fmt.Fprintln(buf, msg.Header.Get("Subject"))
	}
	if _, err := io.Copy(buf, msg.Body); err != nil {
		return nil, fmt.Errorf("read message body: %w", err)
	}
	rcpt, err := msg.Header.AddressList("To")
	if err != nil {
		return nil, fmt.Errorf("parse To field: %w", err)
	}
	if msg.Header.Get("CC") != "" {
		rr, err := msg.Header.AddressList("CC")
		if err != nil {
			return nil, fmt.Errorf("parse CC field: %w", err)
		}
		rcpt = append(rcpt, rr...)
	}
	addrs := make([]string, len(rcpt))
	for i := range rcpt {
		addrs[i] = "@" + rcpt[i].Address
	}
	fmt.Fprintln(buf, strings.Join(addrs, " "))
	post.Content = buf.String()
	return &post, nil
}

func Send(apiRoot, token string, p *Post) error {
	form := make(url.Values)
	if p.InReplyTo != "" {
		form.Set("in_reply_to_id", p.InReplyTo)
	}
	form.Set("status", p.Content)
	req, err := http.NewRequest(http.MethodPost, apiRoot+"/v1/statuses", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("non-ok remote status: %s", resp.Status)
	}
	return nil
}

func Search(apiRoot, token, query string) ([]Post, error) {
	q := make(url.Values)
	q.Set("resolve", "1") // fetch apub objects given as a URL
	q.Set("q", query)
	u, err := url.Parse(apiRoot + "/v2/search")
	if err != nil {
		return nil, err
	}
	u.RawQuery = q.Encode()
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("non-ok response status %s", resp.Status)
	}
	defer resp.Body.Close()
	found := struct {
		Statuses []Post
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&found); err != nil {
		return nil, fmt.Errorf("decode search results: %w", err)
	}
	return found.Statuses, nil
}
