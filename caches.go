package anidb

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"errors"
	"io"
	"log"
	"os"
	"reflect"
	"sync"
)

// Loads caches from the given path.
func LoadCachesFromFile(f string) (err error) {
	fh, err := os.Open(f)

	if err != nil {
		return err
	}
	defer fh.Close()
	return LoadCaches(fh)
}

const cacheMajorVersion = 0
const cacheMinorVersion = 0

type cacheDataVer struct {
	ver  int
	data interface{}
}

type lockable interface {
	Lock()
	Unlock()
}

type rlockable interface {
	lockable
	RLock()
	RUnlock()
}

func getLocks() []lockable {
	return []lockable{
		// caches is special-cased
		&eidAidLock,
		&ed2kFidLock,
		&banTimeLock,
		&titlesFileDataLock,
	}
}

func getCacheData() []cacheDataVer {
	return []cacheDataVer{
		cacheDataVer{0, &titlesFileData},
		cacheDataVer{0, &caches.Caches},
		cacheDataVer{0, &eidAidMap},
		cacheDataVer{0, &ed2kFidMap},
		cacheDataVer{0, &banTime}}
}

// Loads caches from the given io.Reader.
func LoadCaches(r io.Reader) (err error) {
	defer func() { log.Println("Loaded with error", err) }()

	caches.LockAll()        // no defer UnlockAll -- the mutexes get reset
	defer caches.m.Unlock() // but we need to unlock this
	for _, lock := range getLocks() {
		lock.Lock()
		defer lock.Unlock()
	}

	// make sure caches' mutexes are reset even on a decoding failure
	defer func() {
		for _, cache := range caches.Caches {
			cache.m = sync.RWMutex{}
		}
	}()

	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}

	dec := gob.NewDecoder(gz)
	version := 0

	if err = dec.Decode(&version); err != nil {
		return err
	}

	if version != cacheMajorVersion {
		return errors.New("Cache major version mismatch")
	}

	defer func() {
		titlesDB.LoadDB(bytes.NewReader(titlesFileData))

		for _, cache := range caches.Caches {
			cache.intent = make(map[int][]chan Cacheable)
		}
	}()

	version = 0
	for _, v := range append([]cacheDataVer{
		cacheDataVer{0, &version}}, getCacheData()...) {
		if v.ver > version {
			break
		}
		if err = dec.Decode(v.data); err != nil {
			return err
		}
	}

	if version != cacheMinorVersion {
		return errors.New("Cache minor version mismatch")
	}
	return nil
}

// Saves caches to the given path.
func DumpCachesToFile(f string) (err error) {
	fh, err := os.Create(f)
	if err != nil {
		return err
	}
	defer fh.Close()
	return DumpCaches(fh)
}

// Saves caches to the given io.Writer.
//
// The cache is a gzipped, versioned gob of the various internal
// caches.
func DumpCaches(w io.Writer) (err error) {
	defer func() { log.Println("Dumped with error", err) }()

	caches.RLockAll()
	defer caches.RUnlockAll()
	for _, lock := range getLocks() {
		if l, ok := lock.(rlockable); ok {
			l.RLock()
			defer l.RUnlock()
		} else {
			lock.Lock()
			defer lock.Unlock()
		}
	}

	gz, err := gzip.NewWriterLevel(w, gzip.BestCompression)
	if err != nil {
		return err
	}
	defer gz.Close()

	enc := gob.NewEncoder(gz)

	for _, v := range append([]cacheDataVer{
		cacheDataVer{0, cacheMajorVersion},
		cacheDataVer{0, cacheMinorVersion},
	}, getCacheData()...) {
		if err = enc.Encode(v.data); err != nil {
			return err
		}
	}

	return nil
}

type Cacheable interface {
	// Updates the last modified time
	Touch()
	// Returns true if the Cacheable is nil, or if the last modified time is too old.
	IsStale() bool
}

var caches = initCacheMap()

type cacheMap struct {
	m      sync.RWMutex
	Caches map[cacheType]*baseCache
}

type cacheType int

const (
	animeCache = cacheType(iota)
	episodeCache
	groupCache
	fileCache
)

func initCacheMap() *cacheMap {
	return &cacheMap{
		Caches: map[cacheType]*baseCache{
			animeCache:   newBaseCache(),
			episodeCache: newBaseCache(),
			groupCache:   newBaseCache(),
			fileCache:    newBaseCache(),
		},
	}
}

func (c *cacheMap) Get(typ cacheType) *baseCache {
	c.m.RLock()
	defer c.m.RUnlock()

	return c.Caches[typ]
}

func (c *cacheMap) LockAll() {
	c.m.Lock()

	for _, cache := range c.Caches {
		cache.m.Lock()
	}
}
func (c *cacheMap) UnlockAll() {
	c.m.Unlock()

	for _, cache := range c.Caches {
		cache.m.Unlock()
	}
}

func (c *cacheMap) RLockAll() {
	c.m.RLock()

	for _, cache := range c.Caches {
		cache.m.RLock()
	}
}
func (c *cacheMap) RUnlockAll() {
	c.m.RUnlock()

	for _, cache := range c.Caches {
		cache.m.RUnlock()
	}
}

type baseCache struct {
	m      sync.RWMutex
	Cache  map[int]Cacheable
	intent map[int][]chan Cacheable
}

func newBaseCache() *baseCache {
	return &baseCache{
		Cache:  make(map[int]Cacheable),
		intent: make(map[int][]chan Cacheable),
	}
}

func (c *baseCache) Get(id int) Cacheable {
	c.m.RLock()
	defer c.m.RUnlock()

	return c.Cache[id]
}

// Sends the Cacheable to all channels that registered
// Intent and clears the Intent list.
func (c *baseCache) Flush(id int, v Cacheable) {
	c.m.Lock()
	defer c.m.Unlock()

	c._flush(id, v)
}

func (c *baseCache) _flush(id int, v Cacheable) {
	for _, ch := range c.intent[id] {
		ch <- v
		close(ch)
	}
	delete(c.intent, id)
}

// Caches if v is not nil and then Flushes the Intents.
func (c *baseCache) Set(id int, v Cacheable) {
	c.m.Lock()
	defer c.m.Unlock()

	if !reflect.ValueOf(v).IsNil() {
		v.Touch()
		c.Cache[id] = v
	}

	c._flush(id, v)
}

// Register the Intent to get the cache data for this id when
// it's available. Returns false if the caller was the first
// to register it.
func (c *baseCache) Intent(id int, ch chan Cacheable) (ok bool) {
	c.m.Lock()
	defer c.m.Unlock()

	list, ok := c.intent[id]
	c.intent[id] = append(list, ch)

	return ok
}
