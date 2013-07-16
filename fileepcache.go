package anidb

import (
	"github.com/Kovensky/go-fscache"
	"strconv"
	"strings"
	"time"
)

// Gets the Files that the given Group has released for the given
// Episode. Convenience method that calls FilesByGID.
func (adb *AniDB) FilesByGroup(ep *Episode, g *Group) <-chan *File {
	ch := make(chan *File, 1)
	if ep == nil || g == nil {
		ch <- nil
		close(ch)
		return ch
	}
	return adb.FilesByGID(ep, g.GID)
}

// Gets the Files that the Group (given by its ID) has released
// for the given Episode. The returned channel may return multiple
// (or no) Files. Uses the UDP API.
//
// On API error (offline, etc), the first *File returned is nil,
// followed by cached files (which may also be nil).
func (adb *AniDB) FilesByGID(ep *Episode, gid GID) <-chan *File {
	ch := make(chan *File, 10)

	fidChan := adb.FIDsByGID(ep, gid)

	go func() {
		chs := []<-chan *File{}
		for fid := range fidChan {
			chs = append(chs, adb.FileByID(fid))
		}
		for _, c := range chs {
			for f := range c {
				ch <- f
			}
		}
		close(ch)
	}()
	return ch
}

// Gets the FIDs that the Group (given by its ID) has released
// for the given Episode. The returned channel may return multiple
// (or no) FIDs. Uses the UDP API.
//
// On API error (offline, etc), the first *File returned is nil,
// followed by cached files (which may also be nil).
func (adb *AniDB) FIDsByGID(ep *Episode, gid GID) <-chan FID {
	key := []fscache.CacheKey{"fid", "by-eid-gid", ep.EID, gid}

	ch := make(chan FID, 10)

	if ep == nil || gid < 1 {
		ch <- 0
		close(ch)
		return ch
	}

	ic := make(chan notification, 1)
	go func() {
		for c := range ic {
			ch <- c.(FID)
		}
		close(ch)
	}()
	if intentMap.Intent(ic, key...) {
		return ch
	}

	if !Cache.IsValid(InvalidKeyCacheDuration, key...) {
		intentMap.Close(key...)
		return ch
	}

	var fids []FID
	switch ts, err := Cache.Get(&fids, key...); {
	case err == nil && time.Now().Sub(ts) < FileCacheDuration:
		is := intentMap.LockIntent(key...)
		go func() {
			defer intentMap.Free(is, key...)
			defer is.Close()

			for _, fid := range fids {
				is.Notify(fid)
			}
		}()
		return ch
	}

	go func() {
		reply := <-adb.udp.SendRecv("FILE",
			paramMap{
				"aid":   ep.AID,
				"gid":   gid,
				"epno":  ep.Episode.String(),
				"fmask": fileFmask,
				"amask": fileAmask,
			})

		is := intentMap.LockIntent(key...)
		defer intentMap.Free(is, key...)

		switch reply.Code() {
		case 220:
			f := adb.parseFileResponse(reply, true)

			fids = []FID{f.FID}
			CacheSet(&fids, key...)

			cacheFile(f)

			is.NotifyClose(f.FID)
			return
		case 322:
			parts := strings.Split(reply.Lines()[1], "|")
			fids = make([]FID, len(parts))
			for i := range parts {
				id, _ := strconv.ParseInt(parts[i], 10, 32)
				fids[i] = FID(id)
			}

			CacheSet(&fids, key...)
		case 320:
			Cache.SetInvalid(key...)
			is.Close()
			return
		default:
			is.Notify(FID(0))
		}

		defer is.Close()
		for _, fid := range fids {
			is.Notify(fid)
		}
	}()
	return ch
}
