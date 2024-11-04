package lemmy

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type cache struct {
	post      map[int]entry
	community map[string]entry
	mu        *sync.Mutex
}

type entry struct {
	post      Post
	community Community
	expiry    time.Time
}

func (c *cache) store(p Post, com Community, dur time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := time.Now().Add(dur)
	entry := entry{expiry: t}
	if p.Name() != "" {
		entry.post = p
		c.post[p.ID] = entry
	}
	if com.Name() != "" {
		entry.community = com
		c.community[com.Name()] = entry
	}
}

func (c *cache) delete(p Post, com Community) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.post, p.ID)
	delete(c.community, com.Name())
}

// max-age=50
func parseMaxAge(s string) (time.Duration, error) {
	var want string
	elems := strings.Split(s, ",")
	for i := range elems {
		elems[i] = strings.TrimSpace(elems[i])
		if strings.HasPrefix(elems[i], "max-age") {
			want = elems[i]
		}
	}
	_, num, found := strings.Cut(want, "=")
	if !found {
		return 0, fmt.Errorf("missing = separator")
	}
	n, err := strconv.Atoi(num)
	if err != nil {
		return 0, fmt.Errorf("parse seconds: %w", err)
	}
	return time.Duration(n) * time.Second, nil
}

// Cache-Control: public, max-age=50
func extractMaxAge(header http.Header) string {
	cc := header.Get("Cache-Control")
	if !strings.Contains(cc, "max-age=") {
		return ""
	}
	return cc
}
