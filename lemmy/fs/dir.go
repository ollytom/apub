package fs

import (
	"io"
	"io/fs"
)

type dirInfo struct {
	entries []fs.DirEntry
	entryp  int
}

func (d *dirInfo) ReadDir(n int) ([]fs.DirEntry, error) {
	entries := d.entries[d.entryp:]
	if n < 0 {
		d.entryp = len(d.entries) // advance to the end
		if len(entries) == 0 {
			return nil, nil
		}
		return entries, nil
	}

	var err error
	if n >= len(entries) {
		err = io.EOF
	} else if d.entryp >= len(d.entries) {
		err = io.EOF
	} else {
		entries = entries[:n-1]
	}
	d.entryp += n
	return entries, err
}
