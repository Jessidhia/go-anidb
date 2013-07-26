package anidb

import (
	"bytes"
	"github.com/Kovensky/go-anidb/titles"
	"io"
	"net/http"
	"os"
	"time"
)

var titlesDB = &titles.TitlesDatabase{}

// Loads the database from anime-titles.dat.gz in the cache dir.
func RefreshTitles() error {
	if lock, err := Cache.Lock("anime-titles.dat.gz"); err != nil {
		return err
	} else {
		defer lock.Unlock()
	}

	fh, err := Cache.Open("anime-titles.dat.gz")
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

// Returns the number of anime in the titles database
func AnimeCount() int {
	return len(titlesDB.AnimeMap)
}

// Downloads a new anime-titles database if the database is outdated.
//
// Saves the database as anime-titles.dat.gz in the cache dir.
func (adb *AniDB) UpdateTitles() error {
	// needs the AniDB for the Logger

	// too new, no need to update
	if TitlesUpToDate() {
		return nil
	}

	switch lock, err := Cache.Lock("anime-titles.dat.gz"); {
	case os.IsNotExist(err):
		// we're creating it now
	case err == nil:
		defer lock.Unlock()
	default:
		return err
	}

	c := &http.Client{Transport: &http.Transport{DisableCompression: true}}

	adb.Logger.Printf("HTTP>>> %s", titles.DataDumpURL)

	resp, err := c.Get(titles.DataDumpURL)
	if err != nil {
		adb.Logger.Printf("HTTP<<< %s", resp.Status)
		return err
	}
	defer resp.Body.Close()

	buf := bytes.Buffer{}
	adb.Logger.Printf("HTTP--- %s", resp.Status)

	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		adb.Logger.Printf("HTTP--- %v", err)
		return err
	}

	fh, err := Cache.Create("anime-titles.dat.gz")
	if err != nil {
		return err
	}

	_, err = io.Copy(fh, &buf)
	if err != nil {
		return err
	}

	defer func() {
		adb.Logger.Printf("HTTP<<< Titles version %s", titlesDB.UpdateTime)
	}()
	return RefreshTitles()
}
