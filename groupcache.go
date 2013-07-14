package anidb

import (
	"encoding/gob"
	"github.com/Kovensky/go-anidb/http"
	"strconv"
	"strings"
	"time"
)

func init() {
	gob.RegisterName("*github.com/Kovensky/go-anidb.Group", &Group{})
}

func (g *Group) Touch() {
	g.Cached = time.Now()
}

func (g *Group) IsStale() bool {
	if g == nil {
		return true
	}
	return time.Now().Sub(g.Cached) > GroupCacheDuration
}

// Unique Group IDentifier
type GID int

// Retrieves the Group from the cache.
func (gid GID) Group() *Group {
	var g Group
	if cache.Get(&g, "gid", gid) == nil {
		return &g
	}
	return nil
}

// Returns a Group from the cache if possible.
//
// If the Group is stale, then retrieves the Group
// through the UDP API.
func (adb *AniDB) GroupByID(gid GID) <-chan *Group {
	keys := []cacheKey{"gid", gid}
	ch := make(chan *Group, 1)

	ic := make(chan Cacheable, 1)
	go func() { ch <- (<-ic).(*Group); close(ch) }()
	if intentMap.Intent(ic, keys...) {
		return ch
	}

	if g := gid.Group(); !g.IsStale() {
		intentMap.Notify(g, keys...)
		return ch
	}

	go func() {
		reply := <-adb.udp.SendRecv("GROUP",
			paramMap{"gid": gid})

		var g *Group
		if reply.Error() == nil {
			parts := strings.Split(reply.Lines()[1], "|")
			ints := make([]int64, len(parts))
			for i := range parts {
				ints[i], _ = strconv.ParseInt(parts[i], 10, 32)
			}

			irc := ""
			if parts[7] != "" {
				irc = "irc://" + parts[8] + "/" + parts[7][1:]
			}

			pic := ""
			if parts[10] != "" {
				pic = httpapi.AniDBImageBaseURL + parts[10]
			}

			rellist := strings.Split(parts[16], "'")
			relations := make(map[GID]GroupRelationType, len(rellist))
			for _, rel := range rellist {
				r := strings.Split(rel, ",")
				gid, _ := strconv.ParseInt(r[0], 10, 32)
				typ, _ := strconv.ParseInt(r[1], 10, 32)

				relations[GID(gid)] = GroupRelationType(typ)
			}

			ft := time.Unix(ints[11], 0)
			if ints[11] == 0 {
				ft = time.Time{}
			}
			dt := time.Unix(ints[12], 0)
			if ints[12] == 0 {
				dt = time.Time{}
			}
			lr := time.Unix(ints[14], 0)
			if ints[14] == 0 {
				lr = time.Time{}
			}
			la := time.Unix(ints[15], 0)
			if ints[15] == 0 {
				la = time.Time{}
			}

			g = &Group{
				GID: GID(ints[0]),

				Name:      parts[5],
				ShortName: parts[6],

				IRC:     irc,
				URL:     parts[9],
				Picture: pic,

				Founded:   ft,
				Disbanded: dt,
				// ignore ints[13]
				LastRelease:  lr,
				LastActivity: la,

				Rating: Rating{
					Rating:    float32(ints[1]) / 100,
					VoteCount: int(ints[2]),
				},
				AnimeCount: int(ints[3]),
				FileCount:  int(ints[4]),

				RelatedGroups: relations,

				Cached: time.Now(),
			}
		}
		cache.Set(g, keys...)
		intentMap.Notify(g, keys...)
	}()
	return ch
}
