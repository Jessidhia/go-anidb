package misc

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type EpisodeContainer interface {
	// Returns true if this EpisodeContainer is equivalent or a superset of the given EpisodeContainer
	ContainsEpisodes(EpisodeContainer) bool
}

type Formatter interface {
	// Returns a string where the number portion is 0-padded to fit 'width' digits
	Format(width int) string

	// Returns a string where the number portion is 0-padded to be the same length
	// as max.
	FormatLog(max int) string
}

type EpisodeType int

const (
	EpisodeTypeRegular = EpisodeType(1 + iota)
	EpisodeTypeSpecial // "S" episode
	EpisodeTypeCredits // "C" episode
	EpisodeTypeTrailer // "T" episode
	EpisodeTypeParody  // "P" episode
	EpisodeTypeOther   // "O" episode
)

func parseEpisodeType(typ string) EpisodeType {
	switch typ {
	case "":
		return EpisodeTypeRegular
	case "S":
		return EpisodeTypeSpecial
	case "C":
		return EpisodeTypeCredits
	case "T":
		return EpisodeTypeTrailer
	case "P":
		return EpisodeTypeParody
	case "O":
		return EpisodeTypeOther
	}
	return 0
}

func (et EpisodeType) String() string {
	switch et {
	case EpisodeTypeRegular:
		return ""
	case EpisodeTypeSpecial:
		return "S"
	case EpisodeTypeCredits:
		return "C"
	case EpisodeTypeTrailer:
		return "T"
	case EpisodeTypeParody:
		return "P"
	case EpisodeTypeOther:
		return "O"
	default:
		return "!"
	}
}

// An episode (duh).
type Episode struct {
	Type   EpisodeType
	Number int
// returns how many digits are needed to represent this int
func scale(i int) int {
	return 1 + int(math.Floor(math.Log10(float64(i))))
}

// Converts the Episode into AniDB API episode format.
func (ep *Episode) String() string {
	return ep.Format(1)
}

// returns how many digits are needed to represent this episode
func (ep *Episode) scale() int {
	if ep == nil {
		return 1
	}
	return scale(ep.Number)
}

// Returns true if ec is an Episode and is identical to this episode,
// or if ec is a single episode EpisodeRange / EpisodeList that
// contain only this episode.
func (ep *Episode) ContainsEpisodes(ec EpisodeContainer) bool {
	switch e := ec.(type) {
	case *Episode:
		return ep != nil && ep.Type == e.Type && ep.Number == e.Number
	case *EpisodeRange:
	case *EpisodeList:
		return EpisodeList{&EpisodeRange{Type: ep.Type, Start: ep, End: ep}}.ContainsEpisodes(ep)
	default:
	}
	return false
}

func (ep *Episode) Format(width int) string {
	return fmt.Sprintf("%s%0"+strconv.Itoa(width)+"d", ep.Type, ep.Number)
}

func (ep *Episode) FormatLog(max int) string {
	return ep.Format(scale(max))
}

// Parses a string in the usual AniDB API episode format and converts into
// an Episode.
func ParseEpisode(s string) *Episode {
	if no, err := strconv.ParseInt(s, 10, 32); err == nil {
		return &Episode{Type: EpisodeTypeRegular, Number: int(no)}
	} else if len(s) < 1 {
		// s too short
	} else if no, err = strconv.ParseInt(s[1:], 10, 30); err == nil {
		return &Episode{Type: parseEpisodeType(s[:1]), Number: int(no)}
	}
	return nil
}
