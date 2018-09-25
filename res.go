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
)

type ExtractPolicy int

const (
	NoOverwrite ExtractPolicy = iota
	AlwaysOverwrite
	OverwriteIfNewer
	magic = "GRES"
)

func assert(err error) {
	if err != nil {
		panic(err)
	}
}

func copySelf() string {
	fi, err := os.Open(os.Args[0])
	assert(err)
	defer fi.Close()
	fn := os.Args[0] + ".tmp"
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
	fo.Seek(-offset, 2)
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

func extract(path string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()
	assert(os.MkdirAll(path, 0700))
	offset := int64(len(magic) + 4)
	f, err := os.Open(os.Args[0])
	assert(err)
	defer f.Close()
	f.Seek(-offset, 2)
	tag := make([]byte, offset)
	_, err = io.ReadFull(f, tag)
	assert(err)
	if string(tag[:len(magic)]) != magic {
		return errors.New("invalid signature")
	}
	size := binary.BigEndian.Uint32(tag[len(magic):])
	offset += int64(size)
	f.Seek(-offset, 2)
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
	return nil
}

func Extract(path string, policy ExtractPolicy) (err error) {
	err = extract(path)
	return err
}

func Pack(root string) (err error) {
	f, err := ioutil.TempFile("", "gres*.tar.gz")
	assert(err)
	defer func() {
		defer func() {
			f.Close()
		}()
		_, err := f.Seek(0, 0)
		assert(err)
		fn := copySelf()
		g, err := os.OpenFile(fn, os.O_WRONLY|os.O_APPEND, 0755)
		assert(err)
		defer func() {
			err := f.Close()
			if e := recover(); e != nil {
				panic(e)
			}
			assert(err)
			assert(os.Remove(os.Args[0]))
			assert(os.Rename(fn, os.Args[0]))
			assert(os.Chmod(os.Args[0], 0755))
		}()
		n, err := io.Copy(g, f)
		assert(err)
		sig := append([]byte(magic), 0, 0, 0, 0)
		binary.BigEndian.PutUint32(sig[4:], uint32(n))
		_, err = g.Write(sig)
		assert(err)
	}()
	if !strings.HasSuffix(root, "/") {
		root += "/"
	}
	zw, _ := gzip.NewWriterLevel(f, gzip.BestCompression)
	defer func() {
		assert(zw.Close())
	}()
	tw := tar.NewWriter(zw)
	defer func() {
		assert(tw.Close())
	}()
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
	return
}

func Strip() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()
	fn := copySelf()
	assert(os.Remove(os.Args[0]))
	assert(os.Rename(fn, os.Args[0]))
	return
}