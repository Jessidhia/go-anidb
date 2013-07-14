// +build never

package anidb

// Copy&paste this for new cache types
// globally replace: Strut strut SID sid

import (
	"sync"
	"time"
)

type Strut struct {
	Cached time.Time
}

func (v *Strut) touch() {
	v.Cached = time.Now()
}
func (v *Strut) isStale(d time.Duration) bool {
	return time.Now().Sub(v.Cached) > d
}

type SID int

func (sid SID) Strut() *Strut {
	return strutCache.Get(sid)
}

var StrutCacheDuration = DefaultCacheDuration

var strutCache = strutCacheStruct{baseCache: newBaseCache()}

type strutCacheStruct struct{ baseCache }

func (c *strutCacheStruct) Get(id SID) *Strut {
	return c.baseCache.Get(int(id)).(*Strut)
}

func (c *strutCacheStruct) Set(id SID, v *Strut) {
	c.baseCache.Set(int(id), v)
}

func (c *strutCacheStruct) Intent(id SID, ch chan *Strut) (ok bool) {
	ch2 := make(chan cacheable, 1)
	go func() { ch <- (<-ch2).(*Strut) }()
	return c.baseCache.Intent(int(id), ch2)
}

func (adb *AniDB) StrutBySID(id SID) <-chan *Strut {
	ch := make(chan *Strut, 1)
	if v := id.Strut(); !v.isStale(StrutCacheDuration) {
		ch <- v
		close(ch)
		return ch
	}

	if strutCache.Intent(id, ch) {
		return ch
	}

	go func() {
		var v *Strut
		strutCache.Set(id, v)
	}()
	return ch
}
