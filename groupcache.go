package anidb

import (
	"github.com/Kovensky/go-anidb/http"
	"github.com/Kovensky/go-anidb/udp"
	"github.com/Kovensky/go-fscache"
	"strconv"
	"strings"
	"time"
)

var _ cacheable = &Group{}

func (g *Group) setCachedTS(ts time.Time) {
	g.Cached = ts
}

func (g *Group) IsStale() bool {
	if g == nil {
		return true
	}
	return time.Now().Sub(g.Cached) > GroupCacheDuration
}

// Unique Group IDentifier
type GID int

func cacheGroup(g *Group) {
	CacheSet(g.GID, "gid", "by-name", g.Name)
	CacheSet(g.GID, "gid", "by-shortname", g.ShortName)
	CacheSet(g, "gid", g.GID)
}

// Retrieves the Group from the cache.
func (gid GID) Group() *Group {
	var g Group
	if CacheGet(&g, "gid", gid) == nil {
		return &g
	}
	return nil
}

// Retrieves a Group by its GID. Uses the UDP API.
func (adb *AniDB) GroupByID(gid GID) <-chan *Group {
	key := []fscache.CacheKey{"gid", gid}
	ch := make(chan *Group, 1)

	if gid < 1 {
		ch <- nil
		close(ch)
		return ch
	}

	ic := make(chan notification, 1)
	go func() { ch <- (<-ic).(*Group); close(ch) }()
	if intentMap.Intent(ic, key...) {
		return ch
	}

	if !Cache.IsValid(InvalidKeyCacheDuration, key...) {
		intentMap.NotifyClose((*Group)(nil), key...)
		return ch
	}

	g := gid.Group()
	if !g.IsStale() {
		intentMap.NotifyClose(g, key...)
		return ch
	}

	go func() {
		reply := <-adb.udp.SendRecv("GROUP",
			paramMap{"gid": gid})

		if reply.Error() == nil {
			g = parseGroupReply(reply)

			cacheGroup(g)
		} else if reply.Code() == 350 {
			Cache.SetInvalid(key...)
		}

		intentMap.NotifyClose(g, key...)
	}()
	return ch
}

// Retrieves a Group by its name. Either full or short names are matched.
// Uses the UDP API.
func (adb *AniDB) GroupByName(gname string) <-chan *Group {
	key := []fscache.CacheKey{"gid", "by-name", gname}
	altKey := []fscache.CacheKey{"gid", "by-shortname", gname}
	ch := make(chan *Group, 1)

	if gname == "" {
		ch <- nil
		close(ch)
		return ch
	}

	ic := make(chan notification, 1)
	go func() {
		gid := (<-ic).(GID)
		if gid > 0 {
			ch <- <-adb.GroupByID(gid)
		}
		close(ch)
	}()
	if intentMap.Intent(ic, key...) {
		return ch
	}

	if !Cache.IsValid(InvalidKeyCacheDuration, key...) {
		intentMap.NotifyClose(GID(0), key...)
		return ch
	}

	gid := GID(0)

	switch ts, err := Cache.Get(&gid, key...); {
	case err == nil && time.Now().Sub(ts) < GroupCacheDuration:
		intentMap.NotifyClose(gid, key...)
		return ch
	default:
		switch ts, err = Cache.Get(&gid, altKey...); {
		case err == nil && time.Now().Sub(ts) < GroupCacheDuration:
			intentMap.NotifyClose(gid, key...)
			return ch
		}
	}

	go func() {
		reply := <-adb.udp.SendRecv("GROUP",
			paramMap{"gname": gname})

		var g *Group
		if reply.Error() == nil {
			g = parseGroupReply(reply)

			gid = g.GID

			cacheGroup(g)
		} else if reply.Code() == 350 {
			Cache.SetInvalid(key...)
			Cache.SetInvalid(altKey...)
		}

		intentMap.NotifyClose(gid, key...)
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
