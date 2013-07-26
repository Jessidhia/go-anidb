package anidb

import (
	"fmt"
	"github.com/Kovensky/go-anidb/misc"
	"github.com/Kovensky/go-anidb/udp"
	"github.com/Kovensky/go-fscache"
	"image"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var _ cacheable = &File{}

func (f *File) setCachedTS(ts time.Time) {
	f.Cached = ts
}

func (f *File) IsStale() bool {
	if f == nil {
		return true
	}
	if f.Incomplete {
		return time.Now().Sub(f.Cached) > FileIncompleteCacheDuration
	}
	return time.Now().Sub(f.Cached) > FileCacheDuration
}

func cacheFile(f *File) {
	CacheSet(f.FID, "fid", "by-ed2k", f.Ed2kHash, f.Filesize)
	CacheSet(f, "fid", f.FID)
}

type FID int

func (fid FID) File() *File {
	var f File
	if CacheGet(&f, "fid", fid) == nil {
		return &f
	}
	return nil
}

// Prefetches the Anime, Episode and Group that this
// file is linked to using the given AniDB instance.
//
// Returns a channel where this file will be sent to
// when the prefetching is done; if the file is nil,
// the channel will return nil.
func (f *File) Prefetch(adb *AniDB) <-chan *File {
	ch := make(chan *File, 1)
	go func() {
		if f != nil {
			a := adb.AnimeByID(f.AID)
			g := adb.GroupByID(f.GID)
			<-a
			<-g
			ch <- f
		}
		close(ch)
	}()
	return ch
}

// Retrieves a File by its FID. Uses the UDP API.
func (adb *AniDB) FileByID(fid FID) <-chan *File {
	key := []fscache.CacheKey{"fid", fid}

	ch := make(chan *File, 1)

	if fid < 1 {
		ch <- nil
		close(ch)
		return ch
	}

	ic := make(chan notification, 1)
	go func() { ch <- (<-ic).(*File); close(ch) }()
	if intentMap.Intent(ic, key...) {
		return ch
	}

	if !Cache.IsValid(InvalidKeyCacheDuration, key...) {
		intentMap.NotifyClose((*File)(nil), key...)
		return ch
	}

	f := fid.File()
	if !f.IsStale() {
		intentMap.NotifyClose(f, key...)
		return ch
	}

	go func() {
		reply := <-adb.udp.SendRecv("FILE",
			paramMap{
				"fid":   fid,
				"fmask": fileFmask,
				"amask": fileAmask,
			})

		if reply.Error() == nil {
			adb.parseFileResponse(&f, reply, false)

			cacheFile(f)
		} else if reply.Code() == 320 {
			Cache.SetInvalid(key...)
		}

		intentMap.NotifyClose(f, key...)
	}()
	return ch
}

var validEd2kHash = regexp.MustCompile(`\A[[:xdigit:]]{32}\z`)

// Retrieves a File by its Ed2kHash + Filesize combination. Uses the UDP API.
func (adb *AniDB) FileByEd2kSize(ed2k string, size int64) <-chan *File {
	key := []fscache.CacheKey{"fid", "by-ed2k", ed2k, size}

	ch := make(chan *File, 1)

	if size < 1 || !validEd2kHash.MatchString(ed2k) {
		ch <- nil
		close(ch)
		return ch
	}
	// AniDB always uses lower case hashes
	ed2k = strings.ToLower(ed2k)

	ic := make(chan notification, 1)
	go func() {
		fid := (<-ic).(FID)
		if fid > 0 {
			ch <- <-adb.FileByID(fid)
		}
		close(ch)
	}()
	if intentMap.Intent(ic, key...) {
		return ch
	}

	if !Cache.IsValid(InvalidKeyCacheDuration, key...) {
		intentMap.NotifyClose(FID(0), key...)
		return ch
	}

	fid := FID(0)

	switch ts, err := Cache.Get(&fid, key...); {
	case err == nil && time.Now().Sub(ts) < FileCacheDuration:
		intentMap.NotifyClose(fid, key...)
		return ch
	}

	go func() {
		reply := <-adb.udp.SendRecv("FILE",
			paramMap{
				"ed2k":  ed2k,
				"size":  size,
				"fmask": fileFmask,
				"amask": fileAmask,
			})

		var f *File
		if reply.Error() == nil {
			adb.parseFileResponse(&f, reply, false)

			fid = f.FID

			cacheFile(f)
		} else if reply.Code() == 320 { // file not found
			Cache.SetInvalid(key...)
		} else if reply.Code() == 322 { // multiple files found
			panic("Don't know what to do with " + strings.Join(reply.Lines(), "\n"))
		}

		intentMap.NotifyClose(fid, key...)
	}()
	return ch
}

var fileFmask = "7fda7fe8"
var fileAmask = "00008000"

const (
	stateCRCOK = 1 << iota
	stateCRCERR
	stateV2
	stateV3
	stateV4
	stateV5
	stateUncensored
	stateCensored
)

func sanitizeCodec(codec string) string {
	switch codec {
	case "MP3 CBR":
		return "MP3"
	case "WMV9 (also WMV3)":
		return "WMV9"
	case "Ogg (Vorbis)":
		return "Vorbis"
	case "H264/AVC":
		return "H.264"
	}
	return codec
}

var opedRE = regexp.MustCompile(`\A(Opening|Ending)(?: (\d+))?\z`)

func (adb *AniDB) parseFileResponse(f **File, reply udpapi.APIReply, calledFromFIDsByGID bool) bool {
	if reply.Error() != nil {
		return false
	}
	if reply.Truncated() {
		panic("Truncated")
	}

	uidChan := make(chan UID, 1)
	if adb.udp.credentials != nil {
		go func() { uidChan <- <-adb.GetUserUID(decrypt(adb.udp.credentials.username)) }()
	} else {
		uidChan <- 0
		close(uidChan)
	}

	parts := strings.Split(reply.Lines()[1], "|")
	ints := make([]int64, len(parts))
	for i, p := range parts {
		ints[i], _ = strconv.ParseInt(p, 10, 64)
	}

	partial := false

	rels := strings.Split(parts[5], "'")
	relList := make([]EID, 0, len(parts[5]))
	related := make(RelatedEpisodes, len(parts[5]))
	for _, rel := range rels {
		r := strings.Split(rel, ",")
		if len(r) < 2 {
			continue
		}

		eid, _ := strconv.ParseInt(r[0], 10, 32)
		pct, _ := strconv.ParseInt(r[1], 10, 32)
		relList = append(relList, EID(eid))
		related[EID(eid)] = float32(pct) / 100

		if pct != 100 {
			partial = true
		}
	}

	epno := misc.ParseEpisodeList(parts[24])
	fid := FID(ints[0])
	aid := AID(ints[1])
	eid := EID(ints[2])
	gid := GID(ints[3])
	lid := LID(ints[4])

	if !epno[0].Start.ContainsEpisodes(epno[0].End) || len(epno) > 1 || len(relList) > 0 {
		// epno is broken -- we need to sanitize it
		thisEp := <-adb.EpisodeByID(eid)
		bad := false
		if thisEp != nil {
			parts := make([]string, 1, len(relList)+1)
			parts[0] = thisEp.Episode.String()

			// everything after this SHOULD be cache hits now, unless this is somehow
			// linked with an EID from a different anime (*stares at Haruhi*).
			// We don't want to use eps from different AIDs anyway, so that makes
			// the job easier.

			// We check if the related episodes are all in sequence from this one.
			// If they are, we build a new epno with the sequence. Otherwise,
			// our epno will only have the primary episode.

			// gather the episode numbers
			for _, eid := range relList {
				if ep := eid.Episode(); ep != nil && ep.AID == thisEp.AID {
					parts = append(parts, ep.Episode.String())
				} else {
					bad = true
					break
				}
			}

			test := misc.EpisodeList{}
			// only if we didn't break the loop
			if !bad {
				test = misc.ParseEpisodeList(strings.Join(parts, ","))
			}

			if partial {
				if calledFromFIDsByGID {
					epno = test
					adb.Logger.Printf("UDP!!! FID %d is only part of episode %s with no complementary files", fid, epno)
				} else if len(test) == 1 && test[0].Start.Number == test[0].End.Number {
					fids := []int{}

					for fid := range adb.FIDsByGID(thisEp, gid) {
						fids = append(fids, int(fid))
					}
					if len(fids) >= 1 && fids[0] == 0 {
						fids = fids[1:]
						// Only entry was API error
						if len(fids) == 0 {
							return false
						}
					}
					sort.Sort(sort.IntSlice(fids))
					idx := sort.SearchInts(fids, int(fid))
					if idx == len(fids) {
						panic(fmt.Sprintf("FID %d couldn't locate itself", fid))
					}

					epno = test

					// equate pointers
					epno[0].End = epno[0].Start

					epno[0].Start.Parts = len(fids)
					epno[0].Start.Part = idx
				} else {
					panic(fmt.Sprintf("Don't know what to do with partial episode %s (EID %d)", test, eid))
				}
			} else {
				// if they're all in sequence, then we'll only have a single range in the list
				if len(test) == 1 {
					epno = test
				} else {
					// use only the primary epno then
					epno = misc.ParseEpisodeList(thisEp.Episode.String())
				}
			}
		}
	}

	epstr := epno.String()
	if len(epno) == 1 && epno[0].Type == misc.EpisodeTypeCredits && epno[0].Len() == 1 {
		typ := ""
		n := 0

		if ep := <-adb.EpisodeByID(eid); ep == nil {
		} else if m := opedRE.FindStringSubmatch(ep.Titles["en"]); len(m) > 2 {
			num, err := strconv.ParseInt(m[2], 10, 32)
			if err == nil {
				n = int(num)
			}

			typ = m[1]
		}

		gobi := fmt.Sprintf("%d", n)
		if n == 0 {
			gobi = ""
		}

		switch typ {
		case "Opening":
			epstr = "OP" + gobi
		case "Ending":
			epstr = "ED" + gobi
		}
	}

	version := FileVersion(1)
	switch i := ints[7]; {
	case i&stateV5 != 0:
		version = 5
	case i&stateV4 != 0:
		version = 4
	case i&stateV3 != 0:
		version = 3
	case i&stateV2 != 0:
		version = 2
	}

	codecs := strings.Split(parts[14], "'")
	bitrates := strings.Split(parts[15], "'")
	alangs := strings.Split(parts[20], "'")
	streams := make([]AudioStream, len(codecs))
	for i := range streams {
		br, _ := strconv.ParseInt(bitrates[i], 10, 32)
		streams[i] = AudioStream{
			Bitrate:  int(br),
			Codec:    sanitizeCodec(codecs[i]),
			Language: Language(alangs[i]),
		}
	}

	sl := strings.Split(parts[21], "'")
	slangs := make([]Language, len(sl))
	for i := range sl {
		slangs[i] = Language(sl[i])
	}

	depth := int(ints[12])
	if depth == 0 {
		depth = 8
	}
	res := strings.Split(parts[18], "x")
	width, _ := strconv.ParseInt(res[0], 10, 32)
	height, _ := strconv.ParseInt(res[1], 10, 32)
	video := VideoInfo{
		Bitrate:    int(ints[17]),
		Codec:      sanitizeCodec(parts[16]),
		ColorDepth: depth,
		Resolution: image.Rect(0, 0, int(width), int(height)),
	}

	lidMap := LIDMap{}
	if *f != nil {
		lidMap = (*f).LID
	}

	uid := <-uidChan
	if uid != 0 {
		lidMap[uid] = lid
	}

	*f = &File{
		FID: fid,

		AID: aid,
		EID: eid,
		GID: gid,
		LID: lidMap,

		EpisodeString: epstr,
		EpisodeNumber: epno,

		RelatedEpisodes: related,
		Deprecated:      ints[6] != 0,

		CRCMatch:   ints[7]&stateCRCOK != 0,
		BadCRC:     ints[7]&stateCRCERR != 0,
		Version:    version,
		Uncensored: ints[7]&stateUncensored != 0,
		Censored:   ints[7]&stateCensored != 0,

		Incomplete: video.Resolution.Empty(),

		Filesize: ints[8],
		Ed2kHash: parts[9],
		SHA1Hash: parts[10],
		CRC32:    parts[11],

		Source: FileSource(parts[13]),

		AudioStreams:      streams,
		SubtitleLanguages: slangs,
		VideoInfo:         video,
		FileExtension:     parts[19],

		Length:  time.Duration(ints[22]) * time.Second,
		AirDate: time.Unix(ints[23], 0),
	}
	return true
}
