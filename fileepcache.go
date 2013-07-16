package anidb

import (
	"strconv"
	"strings"
	"time"
)

type fidList struct {
	FIDs []FID
	Time time.Time
}

func (l *fidList) Touch()        { l.Time = time.Now() }
func (l *fidList) IsStale() bool { return time.Now().Sub(l.Time) > FileCacheDuration }

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
	keys := []cacheKey{"fid", "by-ep-gid", ep.EID, gid}

	ch := make(chan FID, 10)

	if ep == nil || gid < 1 {
		ch <- 0
		close(ch)
		return ch
	}

	ic := make(chan Cacheable, 1)
	go func() {
		for c := range ic {
			ch <- c.(FID)
		}
		close(ch)
	}()
	if intentMap.Intent(ic, keys...) {
		return ch
	}

	if !cache.CheckValid(keys...) {
		intentMap.Close(keys...)
		return ch
	}

	var fids fidList
	if cache.Get(&fids, keys...) == nil {
		is := intentMap.LockIntent(keys...)
		go func() {
			defer intentMap.Free(is, keys...)
			defer is.Close()

			for _, fid := range fids.FIDs {
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

		is := intentMap.LockIntent(keys...)
		defer intentMap.Free(is, keys...)

		switch reply.Code() {
		case 220:
			f := adb.parseFileResponse(reply, true)

			fids.FIDs = []FID{f.FID}
			cache.Set(&fids, keys...)

			cache.Set(&fidCache{FID: f.FID}, "fid", "by-ed2k", f.Ed2kHash, f.Filesize)
			cache.Set(f, "fid", f.FID)

			is.NotifyClose(f.FID)
			return
		case 322:
			parts := strings.Split(reply.Lines()[1], "|")
			fids.FIDs = make([]FID, len(parts))
			for i := range parts {
				id, _ := strconv.ParseInt(parts[i], 10, 32)
				fids.FIDs[i] = FID(id)
			}

			cache.Set(&fids, keys...)
		case 320:
			cache.MarkInvalid(keys...)
			cache.Delete(keys...)
			is.Close()
			return
		default:
			is.Notify(FID(0))
		}

		defer is.Close()
		for _, fid := range fids.FIDs {
			is.Notify(fid)
		}
	}()
	return ch
}
