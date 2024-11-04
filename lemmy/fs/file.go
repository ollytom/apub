package fs

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"time"

	"olowe.co/apub/lemmy"
)

type fakeStat struct {
	name  string
	size  int64
	mode  fs.FileMode
	mtime time.Time
}

func (s *fakeStat) Name() string       { return s.name }
func (s *fakeStat) Size() int64        { return s.size }
func (s *fakeStat) Mode() fs.FileMode  { return s.mode }
func (s *fakeStat) ModTime() time.Time { return s.mtime }
func (s *fakeStat) IsDir() bool        { return s.mode.IsDir() }
func (s *fakeStat) Sys() any           { return nil }

type dummy struct {
	name     string
	mode     fs.FileMode
	mtime    time.Time
	contents []byte
	dirinfo  *dirInfo
	buf      io.ReadCloser
}

func (f *dummy) Name() string               { return f.name }
func (f *dummy) IsDir() bool                { return f.mode.IsDir() }
func (f *dummy) Type() fs.FileMode          { return f.mode.Type() }
func (f *dummy) Info() (fs.FileInfo, error) { return f.Stat() }

func (f *dummy) Stat() (fs.FileInfo, error) {
	return &fakeStat{
		name:  f.name,
		mode:  f.mode,
		size:  int64(len(f.contents)),
		mtime: f.mtime,
	}, nil
}

func (f *dummy) Read(p []byte) (int, error) {
	if f.buf == nil {
		f.buf = io.NopCloser(bytes.NewReader(f.contents))
	}
	return f.buf.Read(p)
}

func (f *dummy) Close() error {
	if f.buf == nil {
		return nil
	}
	err := f.buf.Close()
	f.buf = nil
	return err
}

func (f *dummy) ReadDir(n int) ([]fs.DirEntry, error) {
	if !f.mode.IsDir() {
		return nil, &fs.PathError{"readdir", f.name, fmt.Errorf("not a directory")}
	} else if f.dirinfo == nil {
		// TODO(otl): is this accidental? maybe return an error here.
		return nil, &fs.PathError{"readdir", f.name, fmt.Errorf("no dirinfo to track reads")}
	}

	return f.dirinfo.ReadDir(n)
}

type lFile struct {
	info    fs.FileInfo
	dirinfo *dirInfo
	client  *lemmy.Client
	buf     io.ReadCloser
}

func (f *lFile) Read(p []byte) (int, error) {
	if f.buf == nil {
		f.buf = io.NopCloser(strings.NewReader("directory"))
	}
	return f.buf.Read(p)
}

func (f *lFile) Close() error {
	if f.buf == nil || f.dirinfo == nil {
		return fs.ErrClosed
	}
	f.dirinfo = nil
	err := f.buf.Close()
	f.buf = nil
	return err
}

func (f *lFile) Stat() (fs.FileInfo, error) {
	return f.info, nil
}

func (f *lFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if f.dirinfo == nil {
		f.dirinfo = new(dirInfo)
		switch f.info.(type) {
		case *lemmy.Community:
			posts, err := f.client.Posts(f.info.Name(), lemmy.ListAll)
			if err != nil {
				return nil, &fs.PathError{"readdir", f.info.Name(), err}
			}
			for _, p := range posts {
				p := p
				f.dirinfo.entries = append(f.dirinfo.entries, fs.FileInfoToDirEntry(&p))
			}
		case *lemmy.Post:
			p := f.info.(*lemmy.Post)
			comments, err := f.client.Comments(p.ID, lemmy.ListAll)
			if err != nil {
				return nil, &fs.PathError{"readdir", f.info.Name(), err}
			}
			for _, c := range comments {
				c := c
				f.dirinfo.entries = append(f.dirinfo.entries, fs.FileInfoToDirEntry(&c))
			}
			f.dirinfo.entries = append(f.dirinfo.entries, postFile(p))
		default:
			return nil, &fs.PathError{"readdir", f.info.Name(), fmt.Errorf("not a directory")}
		}
	}
	return f.dirinfo.ReadDir(n)
}
func postText(p *lemmy.Post) *bytes.Buffer {
	buf := &bytes.Buffer{}
	fmt.Fprintln(buf, "From:", p.CreatorID)
	fmt.Fprintf(buf, "Message-Id: <%d>\n", p.ID)
	fmt.Fprintf(buf, "List-Archive: <%s>\n", p.URL)
	fmt.Fprintln(buf, "Date:", p.ModTime().Format(time.RFC822))
	fmt.Fprintln(buf, "Subject:", p.Title)
	fmt.Fprintln(buf)
	if p.URL != "" {
		fmt.Fprintln(buf, p.URL)
	}
	fmt.Fprintln(buf, p.Body)
	return buf
}

func postFile(p *lemmy.Post) *dummy {
	return &dummy{
		name:     "post",
		mode:     0o444,
		mtime:    p.ModTime(),
		contents: postText(p).Bytes(),
	}
}

func commentText(c *lemmy.Comment) *bytes.Buffer {
	buf := &bytes.Buffer{}
	fmt.Fprintln(buf, "From:", c.CreatorID)
	fmt.Fprintln(buf, "Date:", c.ModTime().Format(time.RFC822))
	fmt.Fprintf(buf, "Message-ID: <%d>\n", c.ID)
	fmt.Fprintf(buf, "List-Archive: <%s>\n", c.ActivityURL)
	fmt.Fprintln(buf, "Subject: Re:", c.PostID)
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, c.Content)
	return buf
}
