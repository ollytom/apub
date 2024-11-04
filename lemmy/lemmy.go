// Package lemmy provides a client interface to the Lemmy HTTP API version 3.
package lemmy

import (
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"time"
)

type Community struct {
	ID        int    `json:"id"`
	FName     string `json:"name"`
	Title     string `json:"title"`
	Local     bool
	ActorID   string `json:"actor_id"`
	Published time.Time
}

func (c *Community) Name() string       { return c.String() }
func (c *Community) Size() int64        { return 0 }
func (c *Community) Mode() fs.FileMode  { return fs.ModeDir | 0o0555 }
func (c *Community) ModTime() time.Time { return c.Published }
func (c *Community) IsDir() bool        { return c.Mode().IsDir() }
func (c *Community) Sys() interface{}   { return nil }

func (c Community) String() string {
	if c.Local {
		return c.FName
	}
	noscheme := strings.TrimPrefix(c.ActorID, "https://")
	instance, _, _ := strings.Cut(noscheme, "/")
	return fmt.Sprintf("%s@%s", c.FName, instance)
}

type Post struct {
	ID        int
	Title     string `json:"name"`
	Body      string
	CreatorID int `json:"creator_id"`
	URL       string
	Published time.Time
	Updated   time.Time
	Creator   Person `json:"-"`
}

func (p *Post) Name() string { return strconv.Itoa(p.ID) }

func (p *Post) Size() int64 {
	return int64(len(p.Body))
}

func (p *Post) Mode() fs.FileMode { return fs.ModeDir | 0o0555 }
func (p *Post) IsDir() bool       { return p.Mode().IsDir() }
func (p *Post) Sys() interface{}  { return nil }
func (p *Post) ModTime() time.Time {
	if p.Updated.IsZero() {
		return p.Published
	}
	return p.Updated
}

type Comment struct {
	ID     int
	PostID int `json:"post_id"`
	// Holds ordered comment IDs referenced by this comment
	// for threading.
	Path        string
	Content     string
	CreatorID   int `json:"creator_id"`
	Published   time.Time
	Updated     time.Time
	ActivityURL string `json:"ap_id"`
	Creator     Person `json:"-"`
}

func (c *Comment) Name() string { return strconv.Itoa(c.ID) }

func (c *Comment) Size() int64       { return 0 }
func (c *Comment) Mode() fs.FileMode { return 0444 }
func (c *Comment) ModTime() time.Time {
	if c.Updated.IsZero() {
		return c.Published
	}
	return c.Updated
}
func (c *Comment) IsDir() bool      { return c.Mode().IsDir() }
func (c *Comment) Sys() interface{} { return nil }

// ParseCommentPath returns the comment IDs referenced by a Comment.
func ParseCommentPath(s string) []int {
	elems := strings.Split(s, ".")
	if len(elems) == 1 {
		return []int{}
	}
	if elems[0] != "0" {
		return []int{}
	}
	refs := make([]int, len(elems))
	for i, ele := range elems {
		id, err := strconv.Atoi(ele)
		if err != nil {
			return refs
		}
		refs[i] = id
	}
	return refs
}

type Person struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	ActorID string `json:"actor_id"`
	Local   bool   `json:"local"`
}

func (p Person) String() string {
	if p.Local {
		return p.Name
	}
	noscheme := strings.TrimPrefix(p.ActorID, "https://")
	instance, _, _ := strings.Cut(noscheme, "/")
	return fmt.Sprintf("%s@%s", p.Name, instance)
}
