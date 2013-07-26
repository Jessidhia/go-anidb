package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/Kovensky/go-anidb"
	ed2khash "github.com/Kovensky/go-ed2k"
	"io"
	"os"
)

var (
	username = flag.String("username", "", "AniDB Username")
	password = flag.String("password", "", "AniDB Password")
	apikey   = flag.String("apikey", "", "UDP API key (optional)")
)

type ProgressReader struct {
	io.Reader

	Prefix  string
	Size    int64
	pos     int64
	prevpos int64
}

func (r *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)

	if r.pos-512*1024 > r.prevpos || r.prevpos == 0 {
		// only every 512KB
		fmt.Printf("%s%.2f%%\r", r.Prefix, float64(r.pos)*100/float64(r.Size))
		r.prevpos = r.pos
	}
	r.pos += int64(n)
	return
}

func (r *ProgressReader) Close() (err error) {
	fmt.Printf("%s%.2f%%\n", r.Prefix, float64(r.pos)*100/float64(r.Size))
	return nil
}

func hashFile(path string) (ed2k string, size int64) {
	fh, err := os.Open(path)
	if err != nil {
		return
	}
	defer fh.Close()

	stat, err := fh.Stat()
	if err != nil {
		return
	}
	size = stat.Size()

	rd := ProgressReader{
		Reader: fh,
		Prefix: fmt.Sprintf("Hashing %s: ", path),
		Size:   size,
	}
	defer rd.Close()

	hash := ed2khash.New(true)
	_, err = io.Copy(hash, &rd)
	if err != nil {
		return
	}

	ed2k = hex.EncodeToString(hash.Sum(nil))
	return
}

func main() {
	flag.Parse()

	if *username == "" || *password == "" {
		fmt.Println("Username and password must be supplied")
		os.Exit(1)
	}

	adb := anidb.NewAniDB()
	adb.SetCredentials(*username, *password, *apikey)
	defer adb.Logout()

	max := len(flag.Args())
	done := make(chan bool, max)

	for _, path := range flag.Args() {
		ed2k, size := hashFile(path)
		if ed2k != "" {
			go func() {
				f := <-adb.FileByEd2kSize(ed2k, size)
				state := anidb.MyListStateHDD
				done <- <-adb.MyListAdd(f, &anidb.MyListSet{State: &state}) != 0
			}()
		} else {
			go func() { done <- false }()
		}
	}

	count := 0
	for ok := range done {
		if ok {
			count++
		}
		max--
		if max == 0 {
			break
		}
	}

	fmt.Println("Added", count, "files to mylist")
}
