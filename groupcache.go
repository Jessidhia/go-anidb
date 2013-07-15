package anidb

import (
	"encoding/gob"
	"github.com/Kovensky/go-anidb/http"
	"github.com/Kovensky/go-anidb/udp"
	"strconv"
	"strings"
	"time"
)

func init() {
	gob.RegisterName("*github.com/Kovensky/go-anidb.Group", &Group{})
	gob.RegisterName("github.com/Kovensky/go-anidb.GID", GID(0))
	gob.RegisterName("*github.com/Kovensky/go-anidb.gidCache", &gidCache{})
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

// make GID cacheable

func (e GID) Touch()        {}
func (e GID) IsStale() bool { return false }

// Retrieves the Group from the cache.
func (gid GID) Group() *Group {
	var g Group
	if cache.Get(&g, "gid", gid) == nil {
		return &g
	}
	return nil
}

type gidCache struct {
	GID
	Time time.Time
}

func (c *gidCache) Touch() { c.Time = time.Now() }
func (c *gidCache) IsStale() bool {
	if c != nil && time.Now().Sub(c.Time) < GroupCacheDuration {
		return false
	}
	return true
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

	if !cache.CheckValid(keys...) {
		intentMap.Notify((*Group)(nil), keys...)
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
			g = parseGroupReply(reply)
		} else if reply.Code() == 350 {
			cache.MarkInvalid(keys...)
		}
		if g != nil {
			cache.Set(&gidCache{GID: g.GID}, "gid", "by-name", g.Name)
			cache.Set(&gidCache{GID: g.GID}, "gid", "by-shortname", g.ShortName)
			cache.Set(g, keys...)
		}

		intentMap.Notify(g, keys...)
	}()
	return ch
}

func (adb *AniDB) GroupByName(gname string) <-chan *Group {
	keys := []cacheKey{"gid", "by-name", gname}
	altKeys := []cacheKey{"gid", "by-shortname", gname}
	ch := make(chan *Group, 1)

	ic := make(chan Cacheable, 1)
	go func() {
		gid := (<-ic).(GID)
		if gid > 0 {
			ch <- <-adb.GroupByID(gid)
		}
		close(ch)
	}()
	if intentMap.Intent(ic, keys...) {
		return ch
	}

	if !cache.CheckValid(keys...) {
		intentMap.Notify(GID(0), keys...)
		return ch
	}

	var gc gidCache
	if cache.Get(&gc, keys...) == nil {
		intentMap.Notify(gc.GID, keys...)
		return ch
	}

	if cache.Get(&gc, altKeys...) == nil {
		intentMap.Notify(gc.GID, keys...)
		return ch
	}

	go func() {
		reply := <-adb.udp.SendRecv("GROUP",
			paramMap{"gname": gname})

		var g *Group
		if reply.Error() == nil {
			g = parseGroupReply(reply)
		} else if reply.Code() == 350 {
			cache.MarkInvalid(keys...)
		}

		gid := GID(0)
		if g != nil {
			gid = g.GID

			cache.Set(&gidCache{GID: gid}, keys...)
			cache.Set(&gidCache{GID: gid}, altKeys...)
			cache.Set(g, "gid", gid)
		}
		intentMap.Notify(gid, keys...)
	}()
	return ch
}

func parseGroupReply(reply udpapi.APIReply) *Group {
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
		if len(r) < 2 {
			continue
		}
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

	return &Group{
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
