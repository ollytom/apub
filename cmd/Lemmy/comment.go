package main

import (
	"fmt"
	"io"
	"net/mail"
	"path"
	"strconv"

	"olowe.co/apub/lemmy"
)

func loadNewReply(pathname string) []byte {
	if pathname == "" {
		return []byte("To: ")
	}
	return []byte(fmt.Sprintf("To: %s\n\n", path.Base(pathname)))
}

func parseReply(r io.Reader) (*lemmy.Comment, error) {
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return nil, err
	}
	var comment lemmy.Comment
	b, err := io.ReadAll(msg.Body)
	if err != nil {
		return nil, err
	}
	comment.Content = string(b)
	if comment.PostID, err = strconv.Atoi(msg.Header.Get("To")); err != nil {
		return nil, fmt.Errorf("parse post id: %w", err)
	}
	return &comment, nil
}

func printThread(w io.Writer, prefix string, parent int, comments []lemmy.Comment) {
	for _, child := range children(parent, comments) {
		fprintComment(w, prefix, child)
		if len(children(child.ID, comments)) > 0 {
			printThread(w, prefix+"\t", child.ID, comments)
		}
	}
}

func fprintComment(w io.Writer, prefix string, c lemmy.Comment) {
	fmt.Fprintln(w, prefix, "From:", c.Creator)
	fmt.Fprintln(w, prefix, "Archived-At:", c.ActivityURL)
	fmt.Fprintln(w, prefix, c.Content)
}

func children(parent int, pool []lemmy.Comment) []lemmy.Comment {
	var kids []lemmy.Comment
	for _, c := range pool {
		refs := lemmy.ParseCommentPath(c.Path)
		pnt := refs[len(refs)-2]
		if pnt == parent {
			kids = append(kids, c)
		}
	}
	return kids
}
