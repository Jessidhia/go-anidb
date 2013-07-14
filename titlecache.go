package anidb

import (
	"bytes"
	"github.com/Kovensky/go-anidb/titles"
	"io"
	"net/http"
	"sync"
	"time"
)

var titlesFileData []byte
var titlesFileDataLock sync.Mutex
var titlesDB = &titles.TitlesDatabase{}

// Loads the anime-titles database from the given io.Reader.
//
// Caches the io.Reader's contents on memory, which gets saved
// by DumpCaches.
func LoadTitles(src io.Reader) error {
	buf := bytes.Buffer{}
	_, err := io.Copy(&buf, src)
	if err != nil && err != io.EOF {
		return err
	}

	titlesFileDataLock.Lock()
	defer titlesFileDataLock.Unlock()

	titlesFileData = buf.Bytes()

	titlesDB.LoadDB(bytes.NewReader(titlesFileData))

	return nil
}

// Saves the currently cached anime-titles database to the given io.Writer.
func (adb *AniDB) SaveCurrentTitles(dst io.Writer) (int64, error) {
	return io.Copy(dst, bytes.NewReader(titlesFileData))
}

// Returns true if the titles database is up-to-date (newer than 24 hours).
func TitlesUpToDate() (ok bool) {
	return time.Now().Sub(titlesDB.UpdateTime) < 24*time.Hour
}

// Downloads a new anime-titles database if the database is outdated.
//
// Caches the contents on memory, which gets saved by DumpCaches.
func UpdateTitles() error {
	// too new, no need to update
	if TitlesUpToDate() {
		return nil
	}

	resp, err := http.Get(titles.DataDumpURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return LoadTitles(resp.Body)
}
