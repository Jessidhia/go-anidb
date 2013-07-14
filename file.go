package anidb

import (
	"fmt"
	"github.com/Kovensky/go-anidb/misc"
	"image"
	"time"
)

type FileVersion int

func (v FileVersion) String() string {
	if v == 1 {
		return ""
	}
	return fmt.Sprintf("v%d", int(v))
}

type FileSource string

type AudioStream struct {
	Codec    string
	Bitrate  int
	Language Language
}

type VideoInfo struct {
	Codec      string
	Bitrate    int
	Resolution image.Rectangle
	ColorDepth int
}

type File struct {
	FID FID

	AID AID
	EID EID
	GID GID

	Incomplete bool

	Deprecated bool
	CRCMatch   bool
	BadCRC     bool
	Version    FileVersion
	Uncensored bool // Meaning unclear, may not be simply !Censored
	Censored   bool // Meaning unclear, may not be simply !Uncensored

	Filesize int64
	Ed2kHash string
	SHA1Hash string
	CRC32    string

	Length  time.Duration
	AirDate time.Time

	AudioStreams      []AudioStream
	SubtitleLanguages []Language
	VideoInfo         VideoInfo
	FileExtension     string

	Source FileSource

	OtherEpisodes misc.EpisodeList

	Cached time.Time
}
