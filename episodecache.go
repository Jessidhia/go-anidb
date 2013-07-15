package anidb

import (
	"encoding/gob"
	"strconv"
	"strings"
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

// Unique Episode IDentifier.
type EID int

// Retrieves the Episode corresponding to this EID from the cache.
func (eid EID) Episode() *Episode {
	var e Episode
	if cache.Get(&e, "eid", eid) == nil {
		return &e
	}
	return nil
}

func cacheEpisode(ep *Episode) {
	cache.Set(ep.AID, "aid", "by-eid", ep.EID)
	cache.Set(ep, "eid", ep.EID)
}

// Retrieves an Episode by its EID.
//
// If we know which AID owns this EID, then it's equivalent
// to an Anime query. Otherwise, uses both the HTTP and UDP
// APIs to retrieve it.
func (adb *AniDB) EpisodeByID(eid EID) <-chan *Episode {
	keys := []cacheKey{"eid", eid}
	ch := make(chan *Episode, 1)

	ic := make(chan Cacheable, 1)
	go func() { ch <- (<-ic).(*Episode); close(ch) }()
	if intentMap.Intent(ic, keys...) {
		return ch
	}

	if !cache.CheckValid(keys...) {
		intentMap.Notify((*Episode)(nil), keys...)
		return ch
	}

	e := eid.Episode()
	if !e.IsStale() {
		intentMap.Notify(e, keys...)
		return ch
	}

	go func() {
		// The UDP API data is worse than the HTTP API anime data,
		// try and get from the corresponding Anime

		aid := AID(0)
		ok := cache.Get(&aid, "aid", "by-eid", eid) == nil

		udpDone := false

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
				} else if reply.Code() == 340 {
					cache.MarkInvalid(keys...)
					cache.Delete(keys...) // deleted EID?
				} else {
					break
				}
				udpDone = true
			}
			a := <-adb.AnimeByID(AID(aid)) // this caches episodes...
			ep := eid.Episode()            // ...so this is now a cache hit

			if !ep.IsStale() {
				e = ep
				break
			} else {
				ok = false

				// check to see if we looked in the right AID
				found := false
				if a != nil {
					for _, ep := range a.Episodes {
						if eid == ep.EID {
							found = true
							break
						}
					}
				}

				// if found, then it's just that the anime is also stale (offline?)
				if found {
					break
				} else {
					// otherwise, the EID<->AID map broke
					cache.Delete("aid", "by-eid", eid)
				}
			}
		}
		intentMap.Notify(e, keys...)
	}()
	return ch
}
