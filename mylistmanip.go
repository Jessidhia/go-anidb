package anidb

import (
	"github.com/Kovensky/go-fscache"
	"strconv"
	"time"
)

// These are all pointers because they're not
// sent at all if they're nil
type MyListSet struct {
	State    *MyListState
	Watched  *bool
	ViewDate *time.Time
	Source   *string
	Storage  *string
	Other    *string
}

func (set *MyListSet) toParamMap() (pm paramMap) {
	pm = paramMap{}
	if set == nil {
		return
	}

	if set.State != nil {
		pm["state"] = *set.State
	}
	if set.Watched != nil {
		pm["viewed"] = *set.Watched
	}
	if set.ViewDate != nil {
		if set.ViewDate.IsZero() {
			pm["viewdate"] = 0
		} else {
			pm["viewdate"] = int(int32(set.ViewDate.Unix()))
		}
	}
	if set.Source != nil {
		pm["source"] = *set.Source
	}
	if set.Storage != nil {
		pm["storage"] = *set.Storage
	}
	if set.Other != nil {
		pm["other"] = *set.Other
	}
	return
}

func (set *MyListSet) update(uid UID, f *File, lid LID) {
	if f.LID[uid] != lid {
		f.LID[uid] = lid
		Cache.Set(f, "fid", f.FID)
		Cache.Chtime(f.Cached, "fid", f.FID)
	}

	mla := uid.MyListAnime(f.AID)
	if mla == nil {
		mla = &MyListAnime{
			EpisodesWithState: MyListStateMap{},
			EpisodesPerGroup:  GroupEpisodes{},
		}
	}
	// We only ever add, not remove -- we don't know if other files also satisfy the list
	eg := mla.EpisodesPerGroup[f.GID]
	eg.Add(f.EpisodeNumber)
	mla.EpisodesPerGroup[f.GID] = eg

	if set.State != nil {
		es := mla.EpisodesWithState[*set.State]
		es.Add(f.EpisodeNumber)
		mla.EpisodesWithState[*set.State] = es
	}

	if set.Watched != nil && *set.Watched ||
		set.ViewDate != nil && !set.ViewDate.IsZero() {
		mla.WatchedEpisodes.Add(f.EpisodeNumber)
	}

	Cache.Set(mla, "mylist-anime", uid, f.AID)
	Cache.Chtime(mla.Cached, "mylist-anime", uid, f.AID)

	if set.ViewDate == nil && set.Watched == nil && set.State == nil &&
		set.Source == nil && set.Storage == nil && set.Other == nil {
		return
	}

	e := lid.MyListEntry()
	if set.ViewDate != nil {
		e.DateWatched = *set.ViewDate
	} else if set.Watched != nil {
		if *set.Watched {
			e.DateWatched = time.Now()
		} else {
			e.DateWatched = time.Time{}
		}
	}
	if set.State != nil {
		e.MyListState = *set.State
	}
	if set.Source != nil {
		e.Source = *set.Source
	}
	if set.Storage != nil {
		e.Storage = *set.Storage
	}
	if set.Other != nil {
		e.Other = *set.Other
	}
	Cache.Set(e, "mylist", lid)
	Cache.Chtime(e.Cached, "mylist", lid)
}

func (adb *AniDB) MyListAdd(f *File, set *MyListSet) <-chan LID {
	ch := make(chan LID, 1)
	if f == nil {
		ch <- 0
		close(ch)
		return ch
	}

	go func() {
		user := <-adb.GetCurrentUser()
		if user == nil || user.UID < 1 {
			ch <- 0
			close(ch)
			return
		}

		// for the intent map; doesn't get cached
		key := []fscache.CacheKey{"mylist-add", user.UID, f.FID}

		ic := make(chan notification, 1)
		go func() { ch <- (<-ic).(LID); close(ch) }()
		if intentMap.Intent(ic, key...) {
			return
		}

		pm := set.toParamMap()
		pm["fid"] = f.FID

		reply := <-adb.udp.SendRecv("MYLISTADD", pm)

		lid := LID(0)

		switch reply.Code() {
		case 310:
			e := adb.parseMylistReply(reply)
			if e != nil {
				lid = e.LID
			}
		case 210:
			id, _ := strconv.ParseInt(reply.Lines()[1], 10, 64)
			lid = LID(id)

			// the 310 case does this in parseMylistReply
			set.update(user.UID, f, lid)
		}

		intentMap.NotifyClose(lid, key...)
	}()

	return ch
}

func (adb *AniDB) MyListAddByEd2kSize(ed2k string, size int64, set *MyListSet) <-chan LID {
	ch := make(chan LID, 1)
	if size < 1 || !validEd2kHash.MatchString(ed2k) {
		ch <- 0
		close(ch)
		return ch
	}

	go func() {
		ch <- <-adb.MyListAdd(<-adb.FileByEd2kSize(ed2k, size), set)
		close(ch)
	}()
	return ch
}

func (adb *AniDB) MyListEdit(f *File, set *MyListSet) <-chan bool {
	ch := make(chan bool, 1)
	if f == nil {
		ch <- false
		close(ch)
		return ch
	}

	go func() {
		user := <-adb.GetCurrentUser()
		if user == nil || user.UID < 1 {
			ch <- false
			close(ch)
			return
		}

		// for the intent map; doesn't get cached
		key := []fscache.CacheKey{"mylist-edit", user.UID, f.FID}

		ic := make(chan notification, 1)
		go func() { ch <- (<-ic).(bool); close(ch) }()
		if intentMap.Intent(ic, key...) {
			return
		}

		pm := set.toParamMap()
		pm["edit"] = 1
		if lid := f.LID[user.UID]; lid > 0 {
			pm["lid"] = lid
		} else {
			pm["fid"] = f.FID
		}

		reply := <-adb.udp.SendRecv("MYLISTADD", pm)

		switch reply.Code() {
		case 311:
			intentMap.NotifyClose(true, key...)

			set.update(user.UID, f, 0)
		default:
			intentMap.NotifyClose(false, key...)
		}
	}()

	return ch
}

func (adb *AniDB) MyListDel(f *File) <-chan bool {
	ch := make(chan bool)
	if f == nil {
		ch <- false
		close(ch)
		return ch
	}

	go func() {
		user := <-adb.GetCurrentUser()
		if user == nil || user.UID < 1 {
			ch <- false
			close(ch)
			return
		}

		// for the intent map; doesn't get cached
		key := []fscache.CacheKey{"mylist-del", user.UID, f.FID}

		ic := make(chan notification, 1)
		go func() { ch <- (<-ic).(bool); close(ch) }()
		if intentMap.Intent(ic, key...) {
			return
		}

		pm := paramMap{}
		if lid := f.LID[user.UID]; lid > 0 {
			pm["lid"] = lid
		} else {
			pm["fid"] = f.FID
		}

		reply := <-adb.udp.SendRecv("MYLISTDEL", pm)

		switch reply.Code() {
		case 211:
			delete(f.LID, user.UID)
			Cache.Set(f, "fid", f.FID)
			Cache.Chtime(f.Cached, "fid", f.FID)

			intentMap.NotifyClose(true, key...)
		default:
			intentMap.NotifyClose(false, key...)
		}
	}()

	return ch
}
