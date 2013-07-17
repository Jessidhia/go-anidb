package anidb

import (
	"encoding/json"
	"fmt"
	"github.com/Kovensky/go-anidb/misc"
	"image"
	"strconv"
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
	LID LIDMap

	EpisodeNumber misc.EpisodeList

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

	// Map of related EIDs to percentages (range 0.0-1.0).
	// The percentage indicates how much of the EID is covered by this file.
	RelatedEpisodes RelatedEpisodes

	Cached time.Time
}

type RelatedEpisodes map[EID]float32

func (er RelatedEpisodes) MarshalJSON() ([]byte, error) {
	generic := make(map[string]float32, len(er))
	for k, v := range er {
		generic[strconv.Itoa(int(k))] = v
	}
	return json.Marshal(generic)
}

func (er RelatedEpisodes) UnmarshalJSON(b []byte) error {
	var generic map[string]float32
	if err := json.Unmarshal(b, &generic); err != nil {
		return err
	}
	for k, v := range generic {
		ik, err := strconv.ParseInt(k, 10, 32)
		if err != nil {
			return err
		}

		er[EID(ik)] = v
	}

	return nil
}

type LIDMap map[UID]LID

func (m LIDMap) MarshalJSON() ([]byte, error) {
	generic := make(map[string]int, len(m))
	for k, v := range m {
		generic[strconv.Itoa(int(k))] = int(v)
	}
	return json.Marshal(generic)
}

func (m LIDMap) UnmarshalJSON(b []byte) error {
	var generic map[string]int
	if err := json.Unmarshal(b, &generic); err != nil {
		return err
	}
	for k, v := range generic {
		ik, err := strconv.ParseInt(k, 10, 32)
		if err != nil {
			return err
		}

		m[UID(ik)] = LID(v)
	}

	return nil
}
