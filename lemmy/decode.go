package lemmy

import (
	"encoding/json"
	"fmt"
	"io"
)

func decodePosts(r io.Reader) ([]Post, error) {
	var jresponse struct {
		Posts []struct {
			Post Post
		}
	}
	if err := json.NewDecoder(r).Decode(&jresponse); err != nil {
		return nil, fmt.Errorf("decode posts response: %w", err)
	}
	var posts []Post
	for _, post := range jresponse.Posts {
		posts = append(posts, post.Post)
	}
	return posts, nil
}

func decodePostResponse(r io.Reader) (Post, Person, Community, error) {
	type jresponse struct {
		PostView struct {
			Post      Post
			Creator   Person
			Community Community
		} `json:"post_view"`
	}
	var jresp jresponse
	if err := json.NewDecoder(r).Decode(&jresp); err != nil {
		return Post{}, Person{}, Community{}, fmt.Errorf("decode post: %w", err)
	}
	jresp.PostView.Post.Creator = jresp.PostView.Creator
	return jresp.PostView.Post, jresp.PostView.Creator, jresp.PostView.Community, nil
}
