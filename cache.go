package anidb

import (
	"github.com/Kovensky/go-fscache"
	"os"
	"path"
	"time"
)

func init() {
	c, err := fscache.NewCacheDir(path.Join(os.TempDir(), "anidb", "cache"))
	if err != nil {
		panic(err)
	}
	Cache = *c

	RefreshTitles()
}

var Cache fscache.CacheDir

type cacheable interface {
	setCachedTS(time.Time)
}

func CacheSet(v interface{}, key ...fscache.CacheKey) (err error) {
	now := time.Now()
	_, err = Cache.Set(v, key...)
	if err != nil {
		return err
	}
	switch t := v.(type) {
	case cacheable:
		t.setCachedTS(now)
	}
	return
}

func CacheGet(v interface{}, key ...fscache.CacheKey) (err error) {
	ts, err := Cache.Get(v, key...)
	if err != nil {
		return err
	}
	switch t := v.(type) {
	case cacheable:
		t.setCachedTS(ts)
	}
	return
}
