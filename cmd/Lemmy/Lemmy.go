package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"9fans.net/go/acme"
	"olowe.co/apub/lemmy"
)

type awin struct {
	*acme.Win
}

func (win *awin) Look(text string) bool {
	if acme.Show(text) != nil {
		return true
	}

	text = strings.TrimSpace(text)
	text = strings.TrimSuffix(text, "/")
	postID, err := strconv.Atoi(text)
	if err != nil {
		return openCommunity(text)
	}

	community := path.Base(win.name())
	return openPost(postID, community)
	return false
}

func (win *awin) Execute(cmd string) bool {
	switch cmd {
	case "Del":
	default:
		log.Println("unsupported execute", cmd)
	}
	return false
}

func (w *awin) name() string {
	buf, err := w.ReadAll("tag")
	if err != nil {
		w.Err(err.Error())
		return ""
	}
	name := strings.Fields(string(buf))[0]
	return path.Clean(name)
}

var client *lemmy.Client

func loadPostList(community string) ([]byte, error) {
	buf := &bytes.Buffer{}
	posts, err := client.Posts(community, lemmy.ListAll)
	if err != nil {
		return buf.Bytes(), err
	}
	for _, p := range posts {
		// 1234/	User
		// 	Hello world!
		// 5678/	Pengguna
		// 	Halo Dunia!
		fmt.Fprintf(buf, "%d/\t%s\n\t%s\n", p.ID, p.Creator, p.Title)
	}
	return buf.Bytes(), err
}

func loadPost(post lemmy.Post) ([]byte, error) {
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "From: %s\n", post.Creator)
	fmt.Fprintf(buf, "Date: %s\n", post.Published.Format(time.RFC822))
	fmt.Fprintf(buf, "Subject: %s\n", post.Title)

	fmt.Fprintln(buf)
	if post.URL != "" {
		fmt.Fprintln(buf, post.URL)
		fmt.Fprintln(buf)
	}
	if post.Body != "" {
		fmt.Fprintln(buf, post.Body)
		fmt.Fprintln(buf)
	}
	return buf.Bytes(), nil
}

func loadComments(id int) ([]byte, error) {
	comments, err := client.Comments(id, lemmy.ListAll)
	if err != nil {
		return nil, err
	}
	buf := &bytes.Buffer{}
	for _, c := range comments {
		refs := lemmy.ParseCommentPath(c.Path)
		// do we have a root comment?
		// A root comment only referenences itself and "0"
		if len(refs) == 2 {
			fprintComment(buf, "", c)
			printThread(buf, "\t", c.ID, comments)
		}
	}
	return buf.Bytes(), nil
}

const Usage string = "usage: Lemmy [host]"

func init() {
	log.SetFlags(0)
	log.SetPrefix("Lemmy: ")
}

func main() {
	debug := flag.Bool("d", false, "enable debug output to stderr")
	login := flag.Bool("l", false, "log in to Lemmy")
	flag.Parse()

	addr := "lemmy.sdf.org"
	if len(flag.Args()) > 1 {
		fmt.Fprintln(os.Stderr, Usage)
		os.Exit(2)
	} else if len(flag.Args()) == 1 {
		addr = flag.Arg(0)
	}
	client = &lemmy.Client{
		Address: addr,
		Debug:   *debug,
	}

	if *login {
		config, err := os.UserConfigDir()
		if err != nil {
			log.Fatalln(err)
		}
		username, password, err := readCreds(path.Join(config, "Lemmy"))
		if err != nil {
			log.Fatalln("read lemmy credentials:", err)
		}
		if err := client.Login(username, password); err != nil {
			log.Fatalln("login:", err)
		}
	}

	openCommunityList()
	acme.AutoExit(true)
	select {}
}

func mustPathMatch(pattern, name string) bool {
	match, err := path.Match(pattern, name)
	if err != nil {
		panic(err)
	}
	return match
}
