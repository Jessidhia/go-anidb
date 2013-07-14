package anidb

import (
	"encoding/gob"
	"fmt"
	"github.com/Kovensky/go-anidb/misc"
	"github.com/Kovensky/go-anidb/udp"
	"image"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

func init() {
	gob.RegisterName("*github.com/Kovensky/go-anidb.File", &File{})
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

func (fid FID) File() *File {
	f, _ := caches.Get(fileCache).Get(int(fid)).(*File)
	return f
}

func ed2kKey(ed2k string, size int64) string {
	return fmt.Sprintf("%s-%016x", ed2k, size)
}

func ed2kCache(f *File) {
	if f != nil {
		ed2kFidLock.Lock()
		defer ed2kFidLock.Unlock()
		ed2kFidMap[ed2kKey(f.Ed2kHash, f.Filesize)] = f.FID
	}
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

var ed2kFidMap = map[string]FID{}
var ed2kIntent = map[string][]chan *File{}
var ed2kFidLock = sync.RWMutex{}

func (adb *AniDB) FileByID(fid FID) <-chan *File {
	ch := make(chan *File, 1)
	if f := fid.File(); !f.IsStale() {
		ch <- f
		close(ch)
		return ch
	}

	fc := caches.Get(fileCache)
	ic := make(chan Cacheable, 1)
	go func() { ch <- (<-ic).(*File); close(ch) }()
	if fc.Intent(int(fid), ic) {
		return ch
	}

	go func() {
		reply := <-adb.udp.SendRecv("FILE",
			paramMap{
				"fid":   fid,
				"fmask": fileFmask,
				"amask": fileAmask,
			})

		var f *File
		if reply.Error() == nil {
			f = parseFileResponse(reply)
		}
		ed2kCache(f)
		fc.Set(int(fid), f)
	}()
	return ch
}

func (adb *AniDB) FileByEd2kSize(ed2k string, size int64) <-chan *File {
	key := ed2kKey(ed2k, size)
	ch := make(chan *File, 1)

	ed2kFidLock.RLock()
	if fid, ok := ed2kFidMap[key]; ok {
		ed2kFidLock.RUnlock()
		if f := fid.File(); f != nil {
			ch <- f
			close(ch)
			return ch
		}
		return adb.FileByID(fid)
	}
	ed2kFidLock.RUnlock()

	ed2kFidLock.Lock()
	if list, ok := ed2kIntent[key]; ok {
		ed2kIntent[key] = append(list, ch)
		return ch
	} else {
		ed2kIntent[key] = append(list, ch)
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
			f = parseFileResponse(reply)

			ed2kCache(f)
			caches.Get(fileCache).Set(int(f.FID), f)
		} else if reply.Code() == 320 { // file not found
			ed2kFidLock.Lock()
			delete(ed2kFidMap, key)
			ed2kFidLock.Unlock()
		} else if reply.Code() == 322 { // multiple files found
			panic("Don't know what to do with " + strings.Join(reply.Lines(), "\n"))
		}

		ed2kFidLock.Lock()
		defer ed2kFidLock.Unlock()

		for _, ch := range ed2kIntent[key] {
			ch <- f
			close(ch)
		}
		delete(ed2kIntent, key)
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

func parseFileResponse(reply udpapi.APIReply) *File {
	if reply.Error() != nil {
		return nil
	}
	if reply.Truncated() {
		panic("Truncated")
	}

	parts := strings.Split(reply.Lines()[1], "|")
	ints := make([]int64, len(parts))
	for i, p := range parts {
		ints[i], _ = strconv.ParseInt(parts[i], 10, 64)
		log.Printf("#%d: %s\n", i, p)
	}

	// how does epno look like?
	log.Println("epno: " + parts[23])

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
		FID: FID(ints[0]),

		AID: AID(ints[1]),
		EID: EID(ints[2]),
		GID: GID(ints[3]),

		OtherEpisodes: misc.ParseEpisodeList(parts[4]).Simplify(),
		Deprecated:    ints[5] != 0,

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
