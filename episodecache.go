package anidb

import (
	"github.com/Kovensky/go-fscache"
	"strconv"
	"strings"
	"time"
)

var _ cacheable = &Episode{}

func (e *Episode) setCachedTS(ts time.Time) {
	e.Cached = ts
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
	if CacheGet(&e, "eid", eid) == nil {
		return &e
	}
	return nil
}

func cacheEpisode(ep *Episode) {
	CacheSet(ep.AID, "aid", "by-eid", ep.EID)
	CacheSet(ep, "eid", ep.EID)
}

// Retrieves an Episode by its EID.
//
// If we know which AID owns this EID, then it's equivalent
// to an Anime query. Otherwise, uses both the HTTP and UDP
// APIs to retrieve it.
func (adb *AniDB) EpisodeByID(eid EID) <-chan *Episode {
	key := []fscache.CacheKey{"eid", eid}
	ch := make(chan *Episode, 1)

	if eid < 1 {
		ch <- nil
		close(ch)
		return ch
	}

	ic := make(chan notification, 1)
	go func() { ch <- (<-ic).(*Episode); close(ch) }()
	if intentMap.Intent(ic, key...) {
		return ch
	}

	if !Cache.IsValid(InvalidKeyCacheDuration, key...) {
		intentMap.NotifyClose((*Episode)(nil), key...)
		return ch
	}

	e := eid.Episode()
	if !e.IsStale() {
		intentMap.NotifyClose(e, key...)
		return ch
	}

	go func() {
		// The UDP API data is worse than the HTTP API anime data and
		// might even get truncated on some pathological cases;
		// try and get from the corresponding Anime, which uses the HTTP
		// API episode list.

		aid := AID(0)
		_, err := Cache.Get(&aid, "aid", "by-eid", eid)
		ok := err == nil

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
					} else {
						break
					}
				} else if reply.Code() == 340 {
					Cache.SetInvalid(key...)
					break
				} else {
					break
				}
				udpDone = true
			}
			a := <-adb.AnimeByID(AID(aid)) // updates the episode cache as well
			ep := a.EpisodeByEID(eid)

			if ep != nil {
				e = ep
				break
			} else {
				// the EID<->AID map broke
				ok = false
				Cache.Delete("aid", "by-eid", eid)
			}
		}
		intentMap.NotifyClose(e, key...)
	}()
	return ch
}
