package anidb

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"reflect"
	"regexp"
	"sync"
	"time"
)

var _ log.Logger

type Cacheable interface {
	// Updates the last modified time
	Touch()
	// Returns true if the Cacheable is nil, or if the last modified time is too old.
	IsStale() bool
}

func init() {
	gob.RegisterName("*github.com/Kovensky/go-anidb.invalidKeyCache", &invalidKeyCache{})
}

type invalidKeyCache struct{ time.Time }

func (c *invalidKeyCache) Touch() {
	c.Time = time.Now()
}
func (c *invalidKeyCache) IsStale() bool {
	return time.Now().Sub(c.Time) > InvalidKeyCacheDuration
}

type cacheDir struct {
	*sync.RWMutex

	CacheDir string
}

func init() {
	if err := SetCacheDir(path.Join(os.TempDir(), "anidb", "cache")); err != nil {
		panic(err)
	}
}

var cache cacheDir

func SetCacheDir(path string) (err error) {
	m := cache.RWMutex
	if m == nil {
		m = &sync.RWMutex{}
		cache.RWMutex = m
	}
	cache.Lock()

	if err = os.MkdirAll(path, 0755|os.ModeDir); err != nil {
		cache.Unlock()
		return err
	}

	cache = cacheDir{
		RWMutex:  m,
		CacheDir: path,
	}

	cache.Unlock()
	RefreshTitles()
	return nil
}

func GetCacheDir() (path string) {
	cache.RLock()
	defer cache.RUnlock()

	return cache.CacheDir
}

type cacheKey interface{}

// All "bad characters" that can't go in Windows paths.
// It's a superset of the "bad characters" on other OSes, so this works.
var badPath = regexp.MustCompile(`[\\/:\*\?\"<>\|]`)

func stringify(stuff ...cacheKey) []string {
	ret := make([]string, len(stuff))
	for i := range stuff {
		s := fmt.Sprint(stuff[i])
		ret[i] = badPath.ReplaceAllLiteralString(s, "_")
	}
	return ret
}

// Each key but the last is treated as a directory.
// The last key is treated as a regular file.
//
// This also means that cache keys that are file-backed
// cannot have subkeys.
func cachePath(keys ...cacheKey) string {
	parts := append([]string{GetCacheDir()}, stringify(keys...)...)
	p := path.Join(parts...)
	return p
}

// Opens the file that backs the specified keys.
func (c *cacheDir) Open(keys ...cacheKey) (fh *os.File, err error) {
	subItem := cachePath(keys...)
	return os.Open(subItem)
}

// Creates a new file to back the specified keys.
func (c *cacheDir) Create(keys ...cacheKey) (fh *os.File, err error) {
	subItem := cachePath(keys...)
	subDir := path.Dir(subItem)

	if err = os.MkdirAll(subDir, 0755|os.ModeDir); err != nil {
		return nil, err
	}
	return os.Create(subItem)
}

// Deletes the file that backs the specified keys.
func (c *cacheDir) Delete(keys ...cacheKey) (err error) {
	return os.Remove(cachePath(keys...))
}

// Deletes the specified key and all subkeys.
func (c *cacheDir) DeleteAll(keys ...cacheKey) (err error) {
	return os.RemoveAll(cachePath(keys...))
}

func (c *cacheDir) Get(v Cacheable, keys ...cacheKey) (err error) {
	defer func() {
		log.Println("Got entry", keys, "(error", err, ")")
	}()
	flock := lockFile(cachePath(keys...))
	flock.Lock()
	defer flock.Unlock()

	fh, err := c.Open(keys...)
	if err != nil {
		return err
	}
	defer func() {
		if e := fh.Close(); err == nil {
			err = e
		}
	}()

	val := reflect.ValueOf(v)
	if k := val.Kind(); k == reflect.Ptr || k == reflect.Interface {
		val = val.Elem()
	}
	if !val.CanSet() {
		// panic because this is an internal coding mistake
		panic("(*cacheDir).Get(): given Cacheable is not setable")
	}
	gz, err := gzip.NewReader(fh)
	if err != nil {
		return err
	}
	defer func() {
		if e := gz.Close(); err == nil {
			err = e
		}
	}()

	// defer func() {
	// 	if err == io.EOF {
	// 		err = nil
	// 	}
	// }()

	switch f := gz.Header.Comment; f {
	case "encoding/gob":
		dec := gob.NewDecoder(gz)
		err = dec.Decode(v)
	default:
		return errors.New(fmt.Sprintf("Cached data (format %q) is not in a known format", f))
	}

	return
}

func (c *cacheDir) Set(v Cacheable, keys ...cacheKey) (n int64, err error) {
	if v := reflect.ValueOf(v); !v.IsValid() {
		panic("reflect.ValueOf() returned invaled value")
	} else if k := v.Kind(); k == reflect.Ptr || k == reflect.Interface {
		if v.IsNil() {
			return // no point in saving nil
		}
	}
	defer func() {
		log.Println("Set entry", keys, "(error", err, ")")
	}()

	// First we encode to memory -- we don't want to create/truncate a file and put bad data in it.
	buf := bytes.Buffer{}
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return 0, err
	}
	gz.Header.Comment = "encoding/gob"

	// it doesn't matter if the caller doesn't see this,
	// the important part is that the cache does.
	v.Touch()

	enc := gob.NewEncoder(gz)
	err = enc.Encode(v)

	if e := gz.Close(); err == nil {
		err = e
	}

	if err != nil {
		return 0, err
	}

	// We have good data, time to actually put it in the cache
	flock := lockFile(cachePath(keys...))
	flock.Lock()
	defer flock.Unlock()

	fh, err := c.Create(keys...)
	if err != nil {
		return 0, err
	}
	defer func() {
		if e := fh.Close(); err == nil {
			err = e
		}
	}()
	n, err = io.Copy(fh, &buf)
	return
}

// Checks if the given keys are not marked as invalid.
//
// If the key was marked as invalid but is no longer considered
// so, deletes the invalid marker.
func (c *cacheDir) CheckValid(keys ...cacheKey) bool {
	invKeys := append([]cacheKey{"invalid"}, keys...)
	inv := invalidKeyCache{}

	if cache.Get(&inv, invKeys...) == nil {
		if inv.IsStale() {
			cache.Delete(invKeys...)
		} else {
			return false
		}
	}
	return true
}

// Deletes the given keys and marks them as invalid.
//
// They are considered invalid for InvalidKeyCacheDuration.
func (c *cacheDir) MarkInvalid(keys ...cacheKey) error {
	invKeys := append([]cacheKey{"invalid"}, keys...)

	cache.Delete(keys...)
	_, err := cache.Set(&invalidKeyCache{}, invKeys...)
	return err
}
