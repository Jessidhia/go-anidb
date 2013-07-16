package anidb

import (
	"encoding/gob"
	"fmt"
	"github.com/Kovensky/go-anidb/misc"
	"github.com/Kovensky/go-anidb/udp"
	"image"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func init() {
	gob.RegisterName("*github.com/Kovensky/go-anidb.File", &File{})
	gob.RegisterName("*github.com/Kovensky/go-anidb.fidCache", &fidCache{})
	gob.RegisterName("github.com/Kovensky/go-anidb.FID", FID(0))
}

func (f *File) Touch() {
	f.Cached = time.Now()
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

type FID int

// make FID Cacheable

func (e FID) Touch()        {}
func (e FID) IsStale() bool { return false }

func (fid FID) File() *File {
	var f File
	if cache.Get(&f, "fid", fid) == nil {
		return &f
	}
	return nil
}

type fidCache struct {
	FID
	Time time.Time
}

func (c *fidCache) Touch() {
	c.Time = time.Now()
}

func (c *fidCache) IsStale() bool {
	return time.Now().Sub(c.Time) > FileCacheDuration
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
	keys := []cacheKey{"fid", fid}

	ch := make(chan *File, 1)

	if fid < 1 {
		ch <- nil
		close(ch)
		return ch
	}

	ic := make(chan Cacheable, 1)
	go func() { ch <- (<-ic).(*File); close(ch) }()
	if intentMap.Intent(ic, keys...) {
		return ch
	}

	if !cache.CheckValid(keys...) {
		intentMap.NotifyClose((*File)(nil), keys...)
		return ch
	}

	f := fid.File()
	if !f.IsStale() {
		intentMap.NotifyClose(f, keys...)
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
			f = adb.parseFileResponse(reply, false)

			cache.Set(&fidCache{FID: f.FID}, "fid", "by-ed2k", f.Ed2kHash, f.Filesize)
			cache.Set(f, keys...)
		} else if reply.Code() == 320 {
			cache.MarkInvalid(keys...)
		}

		intentMap.NotifyClose(f, keys...)
	}()
	return ch
}

var validEd2kHash = regexp.MustCompile(`\A[:xdigit:]{32}\z`)

// Retrieves a File by its Ed2kHash + Filesize combination. Uses the UDP API.
func (adb *AniDB) FileByEd2kSize(ed2k string, size int64) <-chan *File {
	keys := []cacheKey{"fid", "by-ed2k", ed2k, size}

	ch := make(chan *File, 1)

	if size < 1 || !validEd2kHash.MatchString(ed2k) {
		ch <- nil
		close(ch)
		return ch
	}
	// AniDB always uses lower case hashes
	ed2k = strings.ToLower(ed2k)

	ic := make(chan Cacheable, 1)
	go func() {
		fid := (<-ic).(FID)
		if fid > 0 {
			ch <- <-adb.FileByID(fid)
		}
		close(ch)
	}()
	if intentMap.Intent(ic, keys...) {
		return ch
	}

	if !cache.CheckValid(keys...) {
		intentMap.NotifyClose(FID(0), keys...)
		return ch
	}

	fid := FID(0)

	var ec fidCache
	if cache.Get(&ec, keys...) == nil && !ec.IsStale() {
		intentMap.NotifyClose(ec.FID, keys...)
		return ch
	}
	fid = ec.FID

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
			f = adb.parseFileResponse(reply, false)

			fid = f.FID

			cache.Set(&fidCache{FID: fid}, keys...)
			cache.Set(f, "fid", fid)
		} else if reply.Code() == 320 { // file not found
			cache.MarkInvalid(keys...)
		} else if reply.Code() == 322 { // multiple files found
			panic("Don't know what to do with " + strings.Join(reply.Lines(), "\n"))
		}

		intentMap.NotifyClose(fid, keys...)
	}()
	return ch
}

var fileFmask = "77da7fe8"
var fileAmask = "00008000"

const (
	fileStateCRCOK = 1 << iota
	fileStateCRCERR
	fileStateV2
	fileStateV3
	fileStateV4
	fileStateV5
	fileStateUncensored
	fileStateCensored
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

func (adb *AniDB) parseFileResponse(reply udpapi.APIReply, calledFromFIDsByGID bool) *File {
	if reply.Error() != nil {
		return nil
	}
	if reply.Truncated() {
		panic("Truncated")
	}

	parts := strings.Split(reply.Lines()[1], "|")
	ints := make([]int64, len(parts))
	for i, p := range parts {
		ints[i], _ = strconv.ParseInt(p, 10, 64)
	}

	partial := false

	rels := strings.Split(parts[4], "'")
	relList := make([]EID, 0, len(parts[4]))
	related := make(RelatedEpisodes, len(parts[4]))
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

	epno := misc.ParseEpisodeList(parts[23])
	fid := FID(ints[0])
	aid := AID(ints[1])
	eid := EID(ints[2])
	gid := GID(ints[3])

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
					log.Printf("UDP!!! FID %d is only part of episode %s with no complementary files", fid, epno)
				} else if len(test) == 1 && test[0].Start.Number == test[0].End.Number {
					fids := []int{}

					for fid := range adb.FIDsByGID(thisEp, gid) {
						fids = append(fids, int(fid))
					}
					if len(fids) >= 1 && fids[0] == 0 {
						fids = fids[1:]
						// Only entry was API error
						if len(fids) == 0 {
							return nil
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

	version := FileVersion(1)
	switch i := ints[6]; {
	case i&fileStateV5 != 0:
		version = 5
	case i&fileStateV4 != 0:
		version = 4
	case i&fileStateV3 != 0:
		version = 3
	case i&fileStateV2 != 0:
		version = 2
	}

	// codecs (parts[13]), bitrates (ints[14]), langs (parts[19])
	codecs := strings.Split(parts[13], "'")
	bitrates := strings.Split(parts[14], "'")
	alangs := strings.Split(parts[19], "'")
	streams := make([]AudioStream, len(codecs))
	for i := range streams {
		br, _ := strconv.ParseInt(bitrates[i], 10, 32)
		streams[i] = AudioStream{
			Bitrate:  int(br),
			Codec:    sanitizeCodec(codecs[i]),
			Language: Language(alangs[i]),
		}
	}

	sl := strings.Split(parts[20], "'")
	slangs := make([]Language, len(sl))
	for i := range sl {
		slangs[i] = Language(sl[i])
	}

	depth := int(ints[11])
	if depth == 0 {
		depth = 8
	}
	res := strings.Split(parts[17], "x")
	width, _ := strconv.ParseInt(res[0], 10, 32)
	height, _ := strconv.ParseInt(res[1], 10, 32)
	video := VideoInfo{
		Bitrate:    int(ints[16]),
		Codec:      sanitizeCodec(parts[15]),
		ColorDepth: depth,
		Resolution: image.Rect(0, 0, int(width), int(height)),
	}

	return &File{
		FID: fid,

		AID: aid,
		EID: eid,
		GID: gid,

		EpisodeNumber: epno,

		RelatedEpisodes: related,
		Deprecated:      ints[5] != 0,

		CRCMatch:   ints[6]&fileStateCRCOK != 0,
		BadCRC:     ints[6]&fileStateCRCERR != 0,
		Version:    version,
		Uncensored: ints[6]&fileStateUncensored != 0,
		Censored:   ints[6]&fileStateCensored != 0,

		Incomplete: video.Resolution.Empty(),

		Filesize: ints[7],
		Ed2kHash: parts[8],
		SHA1Hash: parts[9],
		CRC32:    parts[10],

		Source: FileSource(parts[12]),

		AudioStreams:      streams,
		SubtitleLanguages: slangs,
		VideoInfo:         video,
		FileExtension:     parts[18],

		Length:  time.Duration(ints[21]) * time.Second,
		AirDate: time.Unix(ints[22], 0),
	}
}
