package lemmy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"sync"
	"time"
)

type Client struct {
	*http.Client
	Address string
	// If true, HTTP request summaries are printed to standard error.
	Debug     bool
	authToken string
	instance  *url.URL
	cache     *cache
	ready     bool
}

type ListMode string

const (
	ListAll        ListMode = "All"
	ListLocal               = "Local"
	ListSubscribed          = "Subscribed"
)

var ErrNotFound error = errors.New("not found")

func (c *Client) init() error {
	if c.Address == "" {
		c.Address = "127.0.0.1"
	}
	if c.instance == nil {
		u, err := url.Parse("https://" + c.Address + "/api/v3/")
		if err != nil {
			return fmt.Errorf("initialise client: parse instance url: %w", err)
		}
		c.instance = u
	}
	if c.Client == nil {
		c.Client = http.DefaultClient
	}
	if c.cache == nil {
		c.cache = &cache{
			post:      make(map[int]entry),
			community: make(map[string]entry),
			mu:        &sync.Mutex{},
		}
	}
	c.ready = true
	return nil
}

func (c *Client) Communities(mode ListMode) ([]Community, error) {
	if !c.ready {
		if err := c.init(); err != nil {
			return nil, err
		}
	}

	params := map[string]string{
		"type_": string(mode),
		"limit": "30", // TODO go through pages
		"sort":  "New",
	}
	if mode == ListSubscribed {
		if c.authToken == "" {
			return nil, errors.New("not logged in, no subscriptions")
		}
		params["auth"] = c.authToken
	}
	resp, err := c.get("community/list", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote status %s: %w", resp.Status, decodeError(resp.Body))
	}

	var response struct {
		Communities []struct {
			Community Community
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode community response: %w", err)
	}
	var communities []Community
	for _, c := range response.Communities {
		communities = append(communities, c.Community)
	}
	return communities, nil
}

func (c *Client) LookupCommunity(name string) (Community, Counts, error) {
	if !c.ready {
		if err := c.init(); err != nil {
			return Community{}, Counts{}, err
		}
	}
	if ent, ok := c.cache.community[name]; ok {
		if time.Now().Before(ent.expiry) {
			return ent.community, Counts{}, nil
		}
		c.cache.delete(ent.post, ent.community)
	}

	params := map[string]string{"name": name}
	resp, err := c.get("community", params)
	if err != nil {
		return Community{}, Counts{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return Community{}, Counts{}, ErrNotFound
	} else if resp.StatusCode != http.StatusOK {
		return Community{}, Counts{}, fmt.Errorf("remote status %s: %w", resp.Status, decodeError(resp.Body))
	}

	type response struct {
		View struct {
			Community Community
			Counts    Counts
		} `json:"community_view"`
	}
	var cres response
	if err := json.NewDecoder(resp.Body).Decode(&cres); err != nil {
		return Community{}, Counts{}, fmt.Errorf("decode community response: %w", err)
	}
	community := cres.View.Community
	age := extractMaxAge(resp.Header)
	if age != "" {
		dur, err := parseMaxAge(age)
		if err != nil {
			return community, Counts{}, fmt.Errorf("parse cache max age from response header: %w", err)
		}
		c.cache.store(Post{}, community, dur)
	}
	return community, cres.View.Counts, nil
}

func (c *Client) Posts(community string, mode ListMode) ([]Post, error) {
	if !c.ready {
		if err := c.init(); err != nil {
			return nil, err
		}
	}

	params := map[string]string{
		"community_name": community,
		//		"limit":          "30",
		"type_": string(mode),
		"sort":  "New",
	}
	resp, err := c.get("post/list", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote status %s: %w", resp.Status, decodeError(resp.Body))
	}
	age := extractMaxAge(resp.Header)
	ttl, err := parseMaxAge(age)
	if c.Debug && err != nil {
		fmt.Fprintln(os.Stderr, "parse cache max-age from header:", err)
	}

	var jresponse struct {
		Posts []struct {
			Post    Post
			Creator Person
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&jresponse); err != nil {
		return nil, fmt.Errorf("decode posts response: %w", err)
	}
	var posts []Post
	for _, post := range jresponse.Posts {
		post.Post.Creator = post.Creator
		posts = append(posts, post.Post)
		if ttl > 0 {
			c.cache.store(post.Post, Community{}, ttl)
		}
	}
	return posts, nil
}

func (c *Client) LookupPost(id int) (Post, error) {
	if !c.ready {
		if err := c.init(); err != nil {
			return Post{}, err
		}
	}
	if ent, ok := c.cache.post[id]; ok {
		if time.Now().Before(ent.expiry) {
			return ent.post, nil
		}
		c.cache.delete(ent.post, Community{})
	}

	params := map[string]string{"id": strconv.Itoa(id)}
	resp, err := c.get("post", params)
	if err != nil {
		return Post{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return Post{}, ErrNotFound
	} else if resp.StatusCode != http.StatusOK {
		return Post{}, fmt.Errorf("remote status %s: %w", resp.Status, decodeError(resp.Body))
	}
	post, _, _, err := decodePostResponse(resp.Body)
	age := extractMaxAge(resp.Header)
	if age != "" {
		dur, err := parseMaxAge(age)
		if err != nil {
			return post, fmt.Errorf("parse cache max age from response header: %w", err)
		}
		c.cache.store(post, Community{}, dur)
	}
	return post, err
}

func (c *Client) Comments(post int, mode ListMode) ([]Comment, error) {
	if !c.ready {
		if err := c.init(); err != nil {
			return nil, err
		}
	}

	params := map[string]string{
		"post_id": strconv.Itoa(post),
		"type_":   string(mode),
		"limit":   "30",
		"sort":    "New",
	}
	resp, err := c.get("comment/list", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote status %s: %w", resp.Status, decodeError(resp.Body))
	}

	var jresponse struct {
		Comments []struct {
			Comment Comment
			Creator Person
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&jresponse); err != nil {
		return nil, fmt.Errorf("decode comments: %w", err)
	}
	var comments []Comment
	for _, comment := range jresponse.Comments {
		comment.Comment.Creator = comment.Creator
		comments = append(comments, comment.Comment)
	}
	return comments, nil
}

func (c *Client) LookupComment(id int) (Comment, error) {
	if !c.ready {
		if err := c.init(); err != nil {
			return Comment{}, err
		}
	}

	params := map[string]string{"id": strconv.Itoa(id)}
	resp, err := c.get("comment", params)
	if err != nil {
		return Comment{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Comment{}, fmt.Errorf("remote status %s: %w", resp.Status, decodeError(resp.Body))
	}

	type jresponse struct {
		CommentView struct {
			Comment Comment
		} `json:"comment_view"`
	}
	var jresp jresponse
	if err := json.NewDecoder(resp.Body).Decode(&jresp); err != nil {
		return Comment{}, fmt.Errorf("decode comment: %w", err)
	}
	return jresp.CommentView.Comment, nil
}

func (c *Client) Reply(post int, parent int, msg string) error {
	if c.authToken == "" {
		return errors.New("not logged in")
	}

	params := map[string]interface{}{
		"post_id": post,
		"content": msg,
		"auth":    c.authToken,
	}
	if parent > 0 {
		params["parent_id"] = strconv.Itoa(parent)
	}
	resp, err := c.post("/comment", params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("remote status %s: %w", resp.Status, decodeError(resp.Body))
	}
	return nil
}

func (c *Client) post(pathname string, params map[string]interface{}) (*http.Response, error) {
	u := *c.instance
	u.Path = path.Join(u.Path, pathname)

	b, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("encode body: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return c.Do(req)
}

func (c *Client) get(pathname string, params map[string]string) (*http.Response, error) {
	u := *c.instance
	u.Path = path.Join(u.Path, pathname)
	vals := make(url.Values)
	for k, v := range params {
		vals.Set(k, v)
	}
	u.RawQuery = vals.Encode()
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if c.Debug {
		fmt.Fprintf(os.Stderr, "%s %s\n", req.Method, req.URL)
	}
	resp, err := c.Do(req)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode == http.StatusServiceUnavailable {
		time.Sleep(2 * time.Second)
		resp, err = c.get(pathname, params)
	}
	return resp, err
}

type jError struct {
	Err string `json:"error"`
}

func (err jError) Error() string { return err.Err }

func decodeError(r io.Reader) error {
	var jerr jError
	if err := json.NewDecoder(r).Decode(&jerr); err != nil {
		return fmt.Errorf("decode error message: %v", err)
	}
	return jerr
}

type Counts struct {
	Posts       int
	Comments    int
	CommunityID int `json:"community_id"`
	PostID      int `json:"post_id"`
}
