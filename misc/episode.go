package misc

import (
	"fmt"
	"strconv"
	"strings"
)

type EpisodeContainer interface {
	// Returns true if this EpisodeContainer is equivalent or a superset of the given EpisodeContainer
	ContainsEpisodes(EpisodeContainer) bool
}

type Formatter interface {
	Format(width int) string
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
}

// Converts the Episode into AniDB API episode format.
func (ep *Episode) String() string {
	return fmt.Sprintf("%s%d", ep.Type, ep.Number)
}

// Returns true if ec is an Episode and is identical to this episode.
func (ep *Episode) ContainsEpisodes(ec EpisodeContainer) bool {
	switch e := ec.(type) {
	case *Episode:
		return ep != nil && ep.Type == e.Type && ep.Number == ep.Number
	default:
	}
	return false
}

func (ep *Episode) Format(width int) string {
	return fmt.Sprintf("%s%0"+strconv.Itoa(width)+"d", ep.Type, ep.Number)
}

// Parses a string in the usual AniDB API episode format and converts into
// an Episode.
//
//	ParseEpisode("1")  <=> &Episode{Type: EpisodeTypeRegular, Number: 1}
//	ParseEpisode("S2") <=> &Episode{Type: EpisodeTypeSpecial, Number: 2}
//	ParseEpisode("03") <=> &Episode{Type: EpisodeTypeRegular, Number: 3}
//	ParseEpisode("")   <=> nil // invalid number
func ParseEpisode(s string) *Episode {
	if no, err := strconv.ParseInt(s, 10, 32); err == nil {
		return &Episode{Type: EpisodeTypeRegular, Number: int(no)}
	} else if no, err = strconv.ParseInt(s[1:], 10, 30); err == nil {
		return &Episode{Type: parseEpisodeType(s[:1]), Number: int(no)}
	}
	return nil
}

// A range of episodes with a start and possibly without an end.
type EpisodeRange struct {
	Type  EpisodeType // Must be equal to both the Start and End types, unless End is nil
	Start *Episode    // The start of the range
	End   *Episode    // The end of the range; may be nil, which represents an endless range
}

// Converts the EpisodeRange into AniDB API range format.
func (ei *EpisodeRange) String() string {
	if ei.End == nil || ei.Start == ei.End || *(ei.Start) == *(ei.End) {
		return ei.Start.String()
	}
	return fmt.Sprintf("%s-%s", ei.Start, ei.End)
}

// If ec is an Episode, returns true if the Episode is of the same type as the range
// and has a Number >= Start.Number; if End is defined, then the episode's Number must
// also be <= End.Number.
//
// If ec is an EpisodeRange, returns true if they are both of the same type and
// the ec's Start.Number is >= this range's Start.Number;
// also returns true if this EpisodeRange is unbounded or if the ec is bounded
// and ec's End.Number is <= this range's End.Number.
//
// Returns false otherwise.
func (er *EpisodeRange) ContainsEpisodes(ec EpisodeContainer) bool {
	if er == nil {
		return false
	}
	if er.Start == nil || er.Start.Type != e.Type ||
		(er.End != nil && er.End.Type != e.Type) {
		panic("Invalid EpisodeRange used")
	}

	switch e := ec.(type) {
	case *Episode:
		if e.Type == er.Type && e.Number >= er.Start.Number {
			if er.End == nil {
				return true
			} else if e.Number <= er.End.Number {
				return true
			}
		}
	case *EpisodeRange:
		if e.Type == er.Type {
			if e.Start.Number >= er.Start.Number {
				if er.End == nil {
					return true
				} else if e.End == nil {
					return false // a finite set can't contain an infinite one
				} else if e.End.Number <= er.End.Number {
					return true
				}
			}
		}
	default:
	}
	return false
}

// Parses a string in the AniDB API range format and converts into an EpisodeRange.
//
//	ParseEpisodeRange("1")     <=> ep := ParseEpisode("1");
//		&EpisodeRange{Type: EpisodeTypeRegular, Start: ep, End: ep}
//	ParseEpisodeRange("S1-")   <=>
//		&EpisodeRange{Type: EpisodeTypeSpecial, Start: ParseEpisode("S1")}
//	ParseEpisodeRange("T1-T3") <=>
//		&EpisodeRange{Type: EpisodeTypeTrailer, Start: ParseEpisode("T1"), End: ParseEpisode("T3")}
//	ParseEpisodeRange("5-S3")  <=> nil // different episode types in range
//	ParseEpisodeRange("")      <=> nil // invalid start of range
func ParseEpisodeRange(s string) *EpisodeRange {
	parts := strings.Split(s, "-")
	if len(parts) > 2 {
		return nil
	}

	eps := [2]*Episode{}
	for i := range parts {
		eps[i] = ParseEpisode(parts[i])
	}
	if eps[0] == nil {
		return nil
	}

	// Not an interval (just "epno") --
	// convert into interval starting and ending in the same episode
	if len(parts) == 1 {
		eps[1] = eps[0]
	}

	if len(parts) > 1 && eps[1] != nil && eps[0].Type != eps[1].Type {
		return nil
	}
	return &EpisodeRange{
		Type:  eps[0].Type,
		Start: eps[0],
		End:   eps[1],
	}
}

type EpisodeList []*EpisodeRange

// Converts the EpisodeList into the AniDB API list format.
func (el EpisodeList) String() string {
	parts := make([]string, len(el))
	for i, er := range el {
		parts[i] = er.String()
	}
	return strings.Join(parts, ",")
}

// Returns true if any of the contained EpisodeRanges contain the
// given EpisodeContainer.
func (el EpisodeList) ContainsEpisodes(ec EpisodeContainer) bool {
	for _, i := range el {
		if i != nil && i.ContainsEpisodes(ec) {
			return true
		}
	}
	return false
}

// Parses a string in the AniDB API list format and converts into
// an EpisodeList.
//
//	ParseEpisodeList("01")       <=> EpisodeList{ParseEpisodeRange("01")}
//	ParseEpisodeList("S2-S3")    <=> EpisodeList{ParseEpisodeRange("S2-S3")}
//	ParseEpisodeList("T1,C1-C3") <=> EpisodeList{ParseEpisodeRange("T1"), ParseEpisodeRange("C1-C3")}
func ParseEpisodeList(s string) (el EpisodeList) {
	parts := strings.Split(s, ",")

	el = make(EpisodeList, len(parts))
	for i := range parts {
		el[i] = ParseEpisodeRange(parts[i])
	}

	return
}
