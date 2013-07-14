package anidb

import (
	"encoding/gob"
	"strconv"
	"strings"
	"sync"
	"time"
)

func init() {
	gob.RegisterName("*github.com/Kovensky/go-anidb.Episode", &Episode{})
}

func (e *Episode) Touch() {
	e.Cached = time.Now()
}

func (e *Episode) IsStale() bool {
	if e == nil {
		return true
	}
	return time.Now().Sub(e.Cached) > EpisodeCacheDuration
}

var eidAidMap = map[EID]AID{}
var eidAidLock = sync.RWMutex{}

// Unique Episode IDentifier.
type EID int

// Retrieves the Episode corresponding to this EID from the cache.
func (eid EID) Episode() *Episode {
	e, _ := caches.Get(episodeCache).Get(int(eid)).(*Episode)
	return e
}

func cacheEpisode(ep *Episode) {
	eidAidLock.Lock()
	defer eidAidLock.Unlock()

	eidAidMap[ep.EID] = ep.AID
	caches.Get(episodeCache).Set(int(ep.EID), ep)
}

// Retrieves the Episode from the cache if possible.
//
// If the result is stale, then queries the UDP API to
// know which AID owns this EID, then gets the episodes
// from the Anime.
func (adb *AniDB) EpisodeByID(eid EID) <-chan *Episode {
	ch := make(chan *Episode, 1)

	if e := eid.Episode(); e != nil && !e.IsStale() {
		ch <- e
		close(ch)
		return ch
	}

	ec := caches.Get(episodeCache)
	ic := make(chan Cacheable, 1)
	go func() { ch <- (<-ic).(*Episode); close(ch) }()
	if ec.Intent(int(eid), ic) {
		return ch
	}

	go func() {
		// The UDP API data is worse than the HTTP API anime data,
		// try and get from the corresponding Anime

		eidAidLock.RLock()
		aid, ok := eidAidMap[eid]
		eidAidLock.RUnlock()

		udpDone := false

		var e *Episode
		for i := 0; i < 2; i++ {
			if !ok && udpDone {
				// couldn't get anime and we already ran the EPISODE query
				break
			}

			if !ok {
				// We don't know what the AID is yet.
				reply := <-adb.udp.SendRecv("EPISODE", paramMap{"eid": eid})

				if reply.Error() == nil {
					parts := strings.Split(reply.Lines()[1], "|")

					if id, err := strconv.ParseInt(parts[1], 10, 32); err == nil {
						ok = true
						aid = AID(id)
					}
				} else {
					break
				}
				udpDone = true
			}
			<-adb.AnimeByID(AID(aid)) // this caches episodes...
			e = eid.Episode()         // ...so this is now a cache hit

			if e != nil {
				break
			} else {
				// if this is somehow still a miss, then the EID<->AID map broke
				eidAidLock.Lock()
				delete(eidAidMap, eid)
				eidAidLock.Unlock()

				ok = false
			}
		}
		// Caching (and channel broadcasting) done by AnimeByID
	}()
	return ch
}
