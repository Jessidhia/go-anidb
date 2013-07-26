package anidb

import (
	"github.com/Kovensky/go-fscache"
	"strconv"
	"strings"
	"time"
)

func (s *MyListStats) setCachedTS(t time.Time) { s.Cached = t }

func (s *MyListStats) IsStale() bool {
	if s == nil || time.Now().Sub(s.Cached) > MyListCacheDuration {
		return true
	}
	return false
}

var _ cacheable = &MyListStats{}

func (u *User) Stats() *MyListStats {
	if u == nil {
		return nil
	}
	var s MyListStats
	if CacheGet(&s, "mylist-stats", u.UID) == nil {
		return &s
	}
	return nil
}

func (adb *AniDB) MyListStats(user *User) <-chan *MyListStats {
	ch := make(chan *MyListStats, 1)
	if user == nil || user.UID < 1 {
		ch <- nil
		close(ch)
		return ch
	}

	key := []fscache.CacheKey{"mylist-stats", user.UID}

	ic := make(chan notification, 1)
	go func() { ch <- (<-ic).(*MyListStats) }()
	if intentMap.Intent(ic, key...) {
		return ch
	}

	stats := user.Stats()
	if !stats.IsStale() {
		defer intentMap.NotifyClose(stats, key...)
		return ch
	}

	go func() {
		if adb.User() == nil {
			r := adb.udp.ReAuth()
			if r.Code() >= 500 {
				intentMap.NotifyClose(stats, key...)
				return
			}
		}

		if user.UID != adb.User().UID {
			intentMap.NotifyClose(stats, key...)
			return
		}

		reply := <-adb.udp.SendRecv("MYLISTSTATS", nil)
		switch reply.Code() {
		case 222:
			parts := strings.Split(reply.Lines()[1], "|")
			ints := make([]int64, len(parts))
			for i := range parts {
				ints[i], _ = strconv.ParseInt(parts[i], 10, 64)
			}

			stats = &MyListStats{
				Anime:    int(ints[0]),
				Episodes: int(ints[1]),
				Files:    int(ints[2]),
				Filesize: ints[3] * 1024 * 1024, // it comes in MB

				AddedAnime:    int(ints[4]),
				AddedEpisodes: int(ints[5]),
				AddedFiles:    int(ints[6]),
				AddedGroups:   int(ints[7]),

				Leech: float32(ints[8]) / 100,
				Glory: float32(ints[9]) / 100,

				ViewedPctDatabase: float32(ints[10]) / 100,
				MyListPctDatabase: float32(ints[11]) / 100,
				// ViewedPctMyList: float32(ints[12]) / 100, // we can calculate a more accurate value
				ViewedEpisodes: int(ints[13]),

				Votes:   int(ints[14]),
				Reviews: int(ints[15]),

				ViewedTime: time.Duration(ints[16]) * time.Minute,
			}
			stats.ViewedPctMyList = float32(stats.ViewedEpisodes) / float32(stats.Episodes)

			if ac := AnimeCount(); ac > 0 {
				stats.AnimePctDatabase = float32(stats.Anime) / float32(ac)
			}

			CacheSet(stats, key...)
		}
		intentMap.NotifyClose(stats, key...)
	}()
	return ch
}
