package res

import (
	"archive/tar"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type (
	ExtractPolicy int
	ExtractFilter func(string) bool
)

const (
	NoOverwrite ExtractPolicy = iota
	OverwriteIfNewer
	AlwaysOverwrite
	Verbatim
	magic = "GRES"
)

func assert(err error) {
	if err != nil {
		panic(err)
	}
}

func copyTarget(target string) string {
	fi, err := os.Open(target)
	assert(err)
	defer fi.Close()
	fn := target + ".tmp"
	fo, err := os.Create(fn)
	assert(err)
	defer func() {
		err := fo.Close()
		if e := recover(); e != nil {
			panic(e)
		}
		assert(err)
	}()
	_, err = io.Copy(fo, fi)
	assert(err)
	offset := int64(len(magic) + 4)
	_, err = fo.Seek(-offset, 2)
	assert(err)
	tag := make([]byte, offset)
	_, err = io.ReadFull(fo, tag)
	assert(err)
	if string(tag[:len(magic)]) == magic {
		st, _ := fi.Stat()
		size := binary.BigEndian.Uint32(tag[len(magic):])
		assert(fo.Truncate(st.Size() - offset - int64(size)))
	}
	return fn
}

func extract(path string, filter ExtractFilter) {
	assert(os.MkdirAll(path, 0700))
	offset := int64(len(magic) + 4)
	exe, err := os.Executable()
	assert(err)
	f, err := os.Open(exe)
	assert(err)
	defer f.Close()
	_, err = f.Seek(-offset, 2)
	assert(err)
	tag := make([]byte, offset)
	_, err = io.ReadFull(f, tag)
	assert(err)
	if string(tag[:len(magic)]) != magic {
		panic(errors.New("invalid signature"))
	}
	size := binary.BigEndian.Uint32(tag[len(magic):])
	offset += int64(size)
	_, err = f.Seek(-offset, 2)
	assert(err)
	zr, err := gzip.NewReader(f)
	assert(err)
	defer zr.Close()
	tr := tar.NewReader(zr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		assert(err)
		if filter != nil && !filter(hdr.Name) {
			continue
		}
		fn := filepath.Join(path, hdr.Name)
		assert(os.MkdirAll(filepath.Dir(fn), 0755))
		func() {
			f, err := os.Create(fn)
			assert(err)
			defer func() {
				assert(f.Close())
				assert(os.Chtimes(fn, hdr.ModTime, hdr.ModTime))
			}()
			_, err = io.Copy(f, tr)
			assert(err)
		}()
	}
}

func Extract(path string, policy ExtractPolicy, filter ...ExtractFilter) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()
	var f ExtractFilter
	if len(filter) > 0 {
		f = filter[0]
	}
	if path == "" || path == "/" {
		panic(errors.New("Extract: path cannot be empty or root (/)"))
	}
	path = filepath.FromSlash(path)
	if policy == Verbatim {
		assert(os.RemoveAll(path))
		extract(path, f)
		return
	}
	tmp := path + ".tmp"
	extract(tmp, f)
	isNewer := func(fn string, t time.Time) (res bool) {
		dst := strings.Replace(fn, tmp, path, 1)
		st, err := os.Stat(dst)
		if err != nil {
			return true
		}
		return t.After(st.ModTime())
	}
	overwrite := func(fn string) {
		dst := strings.Replace(fn, tmp, path, 1)
		os.Remove(dst)
		assert(os.MkdirAll(filepath.Dir(dst), 0700))
		assert(os.Rename(fn, dst))
	}
	assert(filepath.Walk(tmp, func(p string, fi os.FileInfo, e error) error {
		assert(e)
		if fi.IsDir() {
			return nil
		}
		shouldOverwrite := false
		switch policy {
		case NoOverwrite:
			shouldOverwrite = isNewer(p, time.Time{})
		case OverwriteIfNewer:
			shouldOverwrite = isNewer(p, fi.ModTime())
		case AlwaysOverwrite:
			shouldOverwrite = true
		}
		if shouldOverwrite {
			overwrite(p)
		}
		return nil
	}))
	return os.RemoveAll(tmp)
}

func Pack(root, target string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()
	f, err := ioutil.TempFile("", magic+"*.tar.gz")
	assert(err)
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()
	if !strings.HasSuffix(root, "/") {
		root += "/"
	}
	zw, _ := gzip.NewWriterLevel(f, gzip.BestCompression)
	tw := tar.NewWriter(zw)
	assert(filepath.Walk(root, func(p string, fi os.FileInfo, err error) error {
		assert(err)
		if fi.IsDir() || fi.Mode()&os.ModeSymlink == os.ModeSymlink {
			return nil
		}
		f, err := os.Open(p)
		assert(err)
		defer f.Close()
		hdr := &tar.Header{
			Name:    p[len(root):],
			Mode:    0600,
			Size:    fi.Size(),
			ModTime: fi.ModTime(),
		}
		assert(tw.WriteHeader(hdr))
		_, err = io.Copy(tw, f)
		assert(err)
		return nil
	}))
	assert(tw.Close())
	assert(zw.Close())
	_, err = f.Seek(0, 0)
	assert(err)
	if target == "" {
		target = os.Args[0]
	}
	fn := copyTarget(target)
	g, err := os.OpenFile(fn, os.O_WRONLY|os.O_APPEND, 0755)
	assert(err)
	defer func() {
		err := g.Close()
		if e := recover(); e != nil {
			panic(e)
		}
		assert(err)
		assert(os.Remove(target))
		assert(os.Rename(fn, target))
		assert(os.Chmod(target, 0755))
	}()
	n, err := io.Copy(g, f)
	assert(err)
	sig := append([]byte(magic), 0, 0, 0, 0)
	binary.BigEndian.PutUint32(sig[4:], uint32(n))
	_, err = g.Write(sig)
	assert(err)
	return
}
