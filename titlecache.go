package anidb

import (
	"bytes"
	"github.com/Kovensky/go-anidb/titles"
	"io"
	"log"
	"net/http"
	"time"
)

var titlesDB = &titles.TitlesDatabase{}

// Loads the database from anime-titles.dat.gz in the cache dir.
func RefreshTitles() error {
	if flock := lockFile(cachePath("anime-titles.dat.gz")); flock != nil {
		flock.Lock()
		defer flock.Unlock()
	}

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
// Saves the database as anime-titles.dat.gz in the cache dir.
func UpdateTitles() error {
	// too new, no need to update
	if TitlesUpToDate() {
		return nil
	}

	if flock := lockFile(cachePath("anime-titles.dat.gz")); flock != nil {
		flock.Lock()
		defer flock.Unlock()
	}

	c := &http.Client{Transport: &http.Transport{DisableCompression: true}}

	log.Printf("HTTP>>> %s", titles.DataDumpURL)

	resp, err := c.Get(titles.DataDumpURL)
	if err != nil {
		log.Printf("HTTP<<< %s", resp.Status)
		return err
	}
	defer resp.Body.Close()

	buf := bytes.Buffer{}
	log.Printf("HTTP--- %s", resp.Status)

	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		log.Printf("HTTP--- %v", err)
		return err
	}

	fh, err := cache.Create("anime-titles.dat.gz")
	if err != nil {
		return err
	}

	_, err = io.Copy(fh, &buf)
	if err != nil {
		return err
	}

	defer func() {
		log.Printf("HTTP<<< Titles version %s", titlesDB.UpdateTime)
	}()
	return RefreshTitles()
}
