package fs

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strconv"
	"strings"
	"time"

	"olowe.co/apub/lemmy"
)

type FS struct {
	Client  *lemmy.Client
	started bool
}

func (fsys *FS) start() error {
	if fsys.Client == nil {
		fsys.Client = &lemmy.Client{}
	}
	fsys.started = true
	return nil
}

func (fsys *FS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{"open", name, fs.ErrInvalid}
	} else if strings.Contains(name, `\`) {
		return nil, &fs.PathError{"open", name, fs.ErrInvalid}
	}
	name = path.Clean(name)

	if !fsys.started {
		if err := fsys.start(); err != nil {
			return nil, fmt.Errorf("start fs: %w", err)
		}
	}
	if name == "." {
		return fsys.openRoot()
	}

	elems := strings.Split(name, "/")
	// We've only got communities, then posts/comments.
	// Anything deeper cannot exist.
	if len(elems) > 3 {
		return nil, &fs.PathError{"open", name, fs.ErrNotExist}
	}

	community, _, err := fsys.Client.LookupCommunity(elems[0])
	if errors.Is(err, lemmy.ErrNotFound) {
		return nil, &fs.PathError{"open", name, fs.ErrNotExist}
	} else if err != nil {
		return nil, &fs.PathError{"open", name, err}
	}
	if len(elems) == 1 {
		return &lFile{
			info:   &community,
			buf:    io.NopCloser(strings.NewReader(community.Name())),
			client: fsys.Client,
		}, nil
	}

	id, err := strconv.Atoi(elems[1])
	if err != nil {
		return nil, &fs.PathError{"open", name, fmt.Errorf("bad post id")}
	}
	post, err := fsys.Client.LookupPost(id)
	if errors.Is(err, lemmy.ErrNotFound) {
		return nil, &fs.PathError{"open", name, fs.ErrNotExist}
	} else if err != nil {
		return nil, &fs.PathError{"open", name, err}
	}
	if len(elems) == 2 {
		return &lFile{
			info:   &post,
			buf:    io.NopCloser(strings.NewReader(post.Name())),
			client: fsys.Client,
		}, nil
	}
	if elems[2] == "post" {
		info, err := postFile(&post).Stat()
		if err != nil {
			return nil, &fs.PathError{"open", name, fmt.Errorf("prepare post file info: %w", err)}
		}
		return &lFile{
			info:   info,
			buf:    io.NopCloser(postText(&post)),
			client: fsys.Client,
		}, nil
	}

	id, err = strconv.Atoi(elems[2])
	if err != nil {
		return nil, &fs.PathError{"open", name, fmt.Errorf("bad comment id")}
	}
	comment, err := fsys.Client.LookupComment(id)
	if errors.Is(err, lemmy.ErrNotFound) {
		return nil, &fs.PathError{"open", name, fs.ErrNotExist}
	} else if err != nil {
		return nil, &fs.PathError{"open", name, err}
	}
	return &lFile{
		info:   &comment,
		buf:    io.NopCloser(commentText(&comment)),
		client: fsys.Client,
	}, nil
}

func (fsys *FS) openRoot() (fs.File, error) {
	dirinfo := new(dirInfo)
	communities, err := fsys.Client.Communities(lemmy.ListAll)
	if err != nil {
		return nil, err
	}
	for _, c := range communities {
		c := c
		dent := fs.FileInfoToDirEntry(&c)
		dirinfo.entries = append(dirinfo.entries, dent)
	}
	return &dummy{
		name:     ".",
		mode:     fs.ModeDir | 0444,
		contents: []byte("hello, world!"),
		dirinfo:  dirinfo,
		mtime:    time.Now(),
	}, nil
}
