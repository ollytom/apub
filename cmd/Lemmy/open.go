package main

import (
	"errors"
	"log"
	"path"
	"strconv"

	"9fans.net/go/acme"
	"olowe.co/apub/lemmy"
)

func openCommunityList() bool {
	win, err := acme.New()
	if err != nil {
		log.Fatal(err)
	}
	win.Name("/lemmy/")
	win.Ctl("dirty")
	defer win.Ctl("clean")

	communities, err := client.Communities(lemmy.ListAll)
	if err != nil {
		log.Print(err)
		return false
	}
	for _, c := range communities {
		win.Fprintf("body", "%s/\n", c.Name())
	}
	awin := &awin{win}
	go awin.EventLoop(awin)
	return true
}

func openCommunity(name string) bool {
	_, _, err := client.LookupCommunity(name)
	if errors.Is(err, lemmy.ErrNotFound) {
		return false
	} else if err != nil {
		log.Print(err)
		return false
	}

	win, err := acme.New()
	if err != nil {
		log.Fatal(err)
	}
	win.Ctl("dirty")
	defer win.Ctl("clean")

	awin := &awin{win}
	awin.Name(path.Join("/lemmy", name) + "/")

	body, err := loadPostList(name)
	if err != nil {
		win.Err(err.Error())
		return false
	}
	awin.Write("body", body)
	win.Addr("#0")
	win.Ctl("dot=addr")
	win.Ctl("show")
	go awin.EventLoop(awin)
	return true
}

func openPost(id int, community string) bool {
	post, err := client.LookupPost(id)
	if err != nil {
		log.Print(err)
		return false
	}

	win, err := acme.New()
	if err != nil {
		log.Fatal(err)
	}
	awin := &awin{win}
	awin.Name(path.Join("/lemmy", community, strconv.Itoa(id)))
	win.Ctl("dirty")
	defer win.Ctl("clean")

	body, err := loadPost(post)
	if err != nil {
		awin.Err(err.Error())
		return false
	}
	awin.Write("body", body)

	body, err = loadComments(post.ID)
	if err != nil {
		awin.Err(err.Error())
		return false
	}
	awin.Write("body", body)

	win.Addr("#0")
	win.Ctl("dot=addr")
	win.Ctl("show")
	go awin.EventLoop(awin)
	return true
}
