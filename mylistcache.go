package anidb

import (
	"github.com/Kovensky/go-anidb/udp"
	"github.com/Kovensky/go-fscache"
	"strconv"
	"strings"
	"time"
)

func (e *MyListEntry) setCachedTS(ts time.Time) {
	e.Cached = ts
}

func (e *MyListEntry) IsStale() bool {
	if e == nil {
		return true
	}

	max := MyListCacheDuration
	if !e.DateWatched.IsZero() {
		max = MyListWatchedCacheDuration
	}
	return time.Now().Sub(e.Cached) > max
}

var _ cacheable = &MyListEntry{}

func (lid LID) MyListEntry() *MyListEntry {
	var e MyListEntry
	if CacheGet(&e, "lid", lid) == nil {
		return &e
	}
	return nil
}

func (adb *AniDB) MyListByFile(f *File) <-chan *MyListEntry {
	ch := make(chan *MyListEntry, 1)

	if f == nil {
		ch <- nil
		close(ch)
		return ch
	}

	go func() {
		user := <-adb.GetCurrentUser()

		var entry *MyListEntry

		if lid := f.LID[user.UID]; lid != 0 {
			entry = <-adb.MyListByLID(lid)
		}
		if entry == nil {
			entry = <-adb.MyListByFID(f.FID)
		}
		ch <- entry
		close(ch)
	}()
	return ch
}

func (adb *AniDB) MyListByLID(lid LID) <-chan *MyListEntry {
	key := []fscache.CacheKey{"mylist", lid}
	ch := make(chan *MyListEntry, 1)

	if lid < 1 {
		ch <- nil
		close(ch)
		return ch
	}

	ic := make(chan notification, 1)
	go func() { ch <- (<-ic).(*MyListEntry); close(ch) }()
	if intentMap.Intent(ic, key...) {
		return ch
	}

	if !Cache.IsValid(InvalidKeyCacheDuration, key...) {
		intentMap.NotifyClose((*MyListEntry)(nil), key...)
		return ch
	}

	entry := lid.MyListEntry()
	if !entry.IsStale() {
		intentMap.NotifyClose(entry, key...)
		return ch
	}

	go func() {
		reply := <-adb.udp.SendRecv("MYLIST", paramMap{"lid": lid})

		switch reply.Code() {
		case 221:
			entry = adb.parseMylistReply(reply) // caches
		case 312:
			panic("Multiple MYLIST entries when querying for single LID")
		case 321:
			Cache.SetInvalid(key...)
		}

		intentMap.NotifyClose(entry, key...)
	}()
	return ch
}

func (adb *AniDB) MyListByFID(fid FID) <-chan *MyListEntry {
	ch := make(chan *MyListEntry, 1)

	if fid < 1 {
		ch <- nil
		close(ch)
		return ch
	}

	// This is an odd one: we lack enough data at first to create the cache key
	go func() {
		user := <-adb.GetCurrentUser()
		if user == nil || user.UID < 1 {
			ch <- nil
			close(ch)
			return
		}

		key := []fscache.CacheKey{"mylist", "by-fid", fid, user.UID}

		ic := make(chan notification, 1)
		go func() { ch <- (<-ic).(*MyListEntry); close(ch) }()
		if intentMap.Intent(ic, key...) {
			return
		}

		if !Cache.IsValid(InvalidKeyCacheDuration, key...) {
			intentMap.NotifyClose((*MyListEntry)(nil), key...)
			return
		}

		lid := LID(0)
		switch ts, err := Cache.Get(&lid, key...); {
		case err == nil && time.Now().Sub(ts) < LIDCacheDuration:
			intentMap.NotifyClose(<-adb.MyListByLID(lid), key...)
			return
		}

		reply := <-adb.udp.SendRecv("MYLIST", paramMap{"fid": fid})

		var entry *MyListEntry

		switch reply.Code() {
		case 221:
			entry = adb.parseMylistReply(reply) // caches
		case 312:
			panic("Multiple MYLIST entries when querying for single FID")
		case 321:
			Cache.SetInvalid(key...)
		}

		intentMap.NotifyClose(entry, key...)
	}()
	return ch
}

func (adb *AniDB) parseMylistReply(reply udpapi.APIReply) *MyListEntry {
	if reply.Error() != nil {
		return nil
	}

	parts := strings.Split(reply.Lines()[1], "|")
	ints := make([]int64, len(parts))
	for i := range parts {
		ints[i], _ = strconv.ParseInt(parts[i], 10, 64)
	}

	da := time.Unix(ints[5], 0)
	if ints[5] == 0 {
		da = time.Time{}
	}
	dw := time.Unix(ints[7], 0)
	if ints[7] == 0 {
		dw = time.Time{}
	}

	e := &MyListEntry{
		LID: LID(ints[0]),

		FID: FID(ints[1]),
		EID: EID(ints[2]),
		AID: AID(ints[3]),
		GID: GID(ints[4]),

		DateAdded:   da,
		DateWatched: dw,

		State:       FileState(ints[11]),
		MyListState: MyListState(ints[6]),

		Storage: parts[8],
		Source:  parts[9],
		Other:   parts[10],
	}

	user := <-adb.GetCurrentUser()

	if user != nil {
		if f := e.FID.File(); f != nil {
			f.LID[user.UID] = e.LID
			cacheFile(f)
		}

		CacheSet(e, "mylist", "by-fid", e.FID, user.UID)
	}

	// TODO: Add mylist info to Anime, also update there
	CacheSet(e, "mylist", e.LID)

	return e
}
