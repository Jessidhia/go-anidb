package anidb

import (
	"github.com/Kovensky/go-anidb/titles"
	"io"
	"net/http"
	"time"
)

var titlesDB = &titles.TitlesDatabase{}

// Reloads titles from the cache.
func RefreshTitles() error {
	flock := lockFile(cachePath("anime-titles.dat.gz"))
	flock.Lock()
	defer flock.Unlock()

	fh, err := cache.Open("anime-titles.dat.gz")
	if err != nil {
		return err
	}
	defer fh.Close()

	titlesDB.LoadDB(fh)
	return nil
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

	flock := lockFile(cachePath("anime-titles.dat.gz"))
	flock.Lock()
	defer flock.Unlock()

	c := &http.Client{Transport: &http.Transport{DisableCompression: true}}

	resp, err := c.Get(titles.DataDumpURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fh, err := cache.Create("anime-titles.dat.gz")
	if err != nil {
		return err
	}

	_, err = io.Copy(fh, resp.Body)
	if err != nil {
		return err
	}

	return RefreshTitles()
}
