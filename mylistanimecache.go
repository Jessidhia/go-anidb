package anidb

import (
	"github.com/Kovensky/go-anidb/misc"
	"github.com/Kovensky/go-anidb/udp"
	"github.com/Kovensky/go-fscache"
	"strings"
	"time"
)

func (a *MyListAnime) setCachedTS(ts time.Time) {
	a.Cached = ts
}

func (a *MyListAnime) IsStale() bool {
	if a == nil {
		return true
	}

	return time.Now().Sub(a.Cached) > MyListCacheDuration
}

var _ cacheable = &MyListAnime{}

func (uid UID) MyListAnime(aid AID) *MyListAnime {
	var a MyListAnime
	if CacheGet(&a, "mylist-anime", uid, aid) == nil {
		return &a
	}
	return nil
}

func (adb *AniDB) MyListAnime(aid AID) <-chan *MyListAnime {
	ch := make(chan *MyListAnime, 1)

	if aid < 1 {
		ch <- nil
		close(ch)
		return ch
	}

	go func() {
		user := <-adb.GetCurrentUser()
		if user == nil || user.UID < 1 {
			ch <- nil
			close(ch)
			return
		}
		key := []fscache.CacheKey{"mylist-anime", user.UID, aid}

		ic := make(chan notification, 1)
		go func() { ch <- (<-ic).(*MyListAnime); close(ch) }()
		if intentMap.Intent(ic, key...) {
			return
		}

		if !Cache.IsValid(InvalidKeyCacheDuration, key...) {
			intentMap.NotifyClose((*MyListAnime)(nil), key...)
			return
		}

		entry := user.UID.MyListAnime(aid)
		if !entry.IsStale() {
			intentMap.NotifyClose(entry, key...)
			return
		}

		reply := <-adb.udp.SendRecv("MYLIST", paramMap{"aid": aid})

		switch reply.Code() {
		case 221:
			r := adb.parseMylistReply(reply) // caches

			// we have only a single file added for this anime -- construct a fake 312 struct
			entry = &MyListAnime{AID: aid}

			ep := <-adb.EpisodeByID(r.EID)
			list := misc.EpisodeToList(&ep.Episode)

			entry.EpisodesWithState = MyListStateMap{
				r.MyListState: list,
			}

			if !r.DateWatched.IsZero() {
				entry.WatchedEpisodes = list
			}

			entry.EpisodesPerGroup = GroupEpisodes{
				r.GID: list,
			}
		case 312:
			entry = adb.parseMylistAnime(reply)
			entry.AID = aid
		case 321:
			Cache.SetInvalid(key...)
		}

		CacheSet(entry, key...)
		intentMap.NotifyClose(entry, key...)
	}()
	return ch
}

func (adb *AniDB) UserMyListAnime(uid UID, aid AID) <-chan *MyListAnime {
	key := []fscache.CacheKey{"mylist-anime", uid, aid}
	ch := make(chan *MyListAnime, 1)

	if uid < 1 || aid < 1 {
		ch <- nil
		close(ch)
		return ch
	}

	ic := make(chan notification, 1)
	go func() { ch <- (<-ic).(*MyListAnime); close(ch) }()
	if intentMap.Intent(ic, key...) {
		return ch
	}

	if !Cache.IsValid(InvalidKeyCacheDuration, key...) {
		intentMap.NotifyClose((*MyListAnime)(nil), key...)
		return ch
	}

	entry := uid.MyListAnime(aid)
	if !entry.IsStale() {
		intentMap.NotifyClose(entry, key...)
		return ch
	}

	go func() {
		user := <-adb.GetCurrentUser()

		if user.UID != uid { // we can't query other users' lists from API
			intentMap.NotifyClose(entry, key...)
			return
		}

		intentMap.NotifyClose(<-adb.MyListAnime(aid), key...)
	}()
	return ch
}

func (adb *AniDB) parseMylistAnime(reply udpapi.APIReply) *MyListAnime {
	if reply.Code() != 312 {
		return nil
	}

	parts := strings.Split(reply.Lines()[1], "|")

	// Everything from index 7 on is pairs of group name on odd positions and episode list on even
	var groupParts []string
	if len(parts) > 7 {
		groupParts = parts[7:]
	}

	groupMap := make(GroupEpisodes, len(groupParts)/2)

	for i := 0; i+1 < len(groupParts); i += 2 {
		g := <-adb.GroupByName(groupParts[i])
		if g == nil {
			continue
		}

		groupMap[g.GID] = misc.ParseEpisodeList(groupParts[i+1])
	}

	return &MyListAnime{
		EpisodesWithState: MyListStateMap{
			MyListStateUnknown: misc.ParseEpisodeList(parts[2]),
			MyListStateHDD:     misc.ParseEpisodeList(parts[3]),
			MyListStateCD:      misc.ParseEpisodeList(parts[4]),
			MyListStateDeleted: misc.ParseEpisodeList(parts[5]),
		},

		WatchedEpisodes: misc.ParseEpisodeList(parts[6]),

		EpisodesPerGroup: groupMap,
	}
}
