package mastodon

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type notification struct {
	Post Post `json:"status"`
}

type Stream struct {
	rc   io.ReadCloser
	sc   *bufio.Scanner
	err  error
	post Post
}

type field struct {
	Name string
	Data []byte
}

func parseField(line []byte) field {
	name, data, found := bytes.Cut(line, []byte(":"))
	if !found {
		return field{}
	}
	return field{string(name), bytes.TrimSpace(data)}
}

func (st *Stream) Next() bool {
	if !st.sc.Scan() {
		return false
	}
	if st.sc.Text() == "" {
		return st.Next()
	} else if strings.HasPrefix(st.sc.Text(), ":") {
		// comment message; safe to ignore
		return st.Next()
	}
	ev := parseField(st.sc.Bytes())
	switch ev.Name {
	default:
		st.err = fmt.Errorf("invalid field in message: %s", ev.Name)
	case "event":
		if string(ev.Data) != "update" {
			st.err = fmt.Errorf("unhandled event type %s", string(ev.Data))
			st.rc.Close()
			return false
		}
		return st.Next()
	case "data":
		var p Post
		if err := json.Unmarshal(ev.Data, &p); err != nil {
			st.err = err
			st.rc.Close()
			return false
		}
		st.post = p
		return true
	}
	st.rc.Close()
	return false
}

func (st *Stream) Post() Post {
	return st.post
}

func (st *Stream) Err() error {
	return st.err
}

func Watch(streamURL, token string) (*Stream, error) {
	req, err := http.NewRequest(http.MethodGet, streamURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("non-ok response status: %s", resp.Status)
	}
	return &Stream{
		rc: resp.Body,
		sc: bufio.NewScanner(resp.Body),
	}, nil
}
