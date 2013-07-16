package misc

import (
	"fmt"
	"strings"
)

// A range of episodes with a start and possibly without an end.
type EpisodeRange struct {
	Type  EpisodeType // Must be equal to both the Start and End types; if End is nil, must be equal to the Start type
	Start *Episode    // The start of the range
	End   *Episode    // The end of the range; may be nil, which represents an endless range
}

// Converts the EpisodeRange into AniDB API range format.
func (er *EpisodeRange) String() string {
	return er.Format(er.scale())
}

func (er *EpisodeRange) Format(width int) string {
	if er.Start == er.End || (er.End != nil && *(er.Start) == *(er.End)) {
		return er.Start.Format(width)
	}

	if er.End == nil {
		return fmt.Sprintf("%s-", er.Start.Format(width))
	}
	return fmt.Sprintf("%s-%s", er.Start.Format(width), er.End.Format(width))
}

func (er *EpisodeRange) FormatLog(max int) string {
	return er.Format(scale(max))
}

func (er *EpisodeRange) scale() int {
	if er == nil {
		return 1
	}
	s, e := er.Start.scale(), er.End.scale()
	if e > s {
		return e
	}
	return s
}

// If ec is an *Episode, returns true if the Episode is of the same type as the range
// and has a Number >= Start.Number; if End is defined, then the episode's Number must
// also be <= End.Number.
//
// If ec is an *EpisodeRange, returns true if they are both of the same type and
// the ec's Start.Number is >= this range's Start.Number;
// also returns true if this EpisodeRange is unbounded or if the ec is bounded
// and ec's End.Number is <= this range's End.Number.
//
// If ec is an EpisodeList, returns true if all listed EpisodeRanges are contained
// by this EpisodeRange.
//
// Returns false otherwise.
func (er *EpisodeRange) ContainsEpisodes(ec EpisodeContainer) bool {
	if er == nil {
		return false
	}
	if er.Start == nil || er.Start.Type != er.Type ||
		(er.End != nil && er.End.Type != er.Type) {
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
	case EpisodeList:
		for _, ec := range e {
			if !er.ContainsEpisodes(ec) {
				return false
			}
		}
		return true
	default:
	}
	return false
}

// Tries to merge a with b, returning a new *EpisodeRange that's
// a superset of both a and b.
//
// Returns nil if a and b don't intersect, or are not adjacent.
func (a *EpisodeRange) Merge(b *EpisodeRange) (c *EpisodeRange) {
	if a.touches(b) {
		c = &EpisodeRange{Type: a.Type}

		if a.Start.Number <= b.Start.Number {
			c.Start = a.Start
		} else {
			c.Start = b.Start
		}

		switch {
		case a.End == nil || b.End == nil:
			c.End = nil
		case a.End.Number >= b.End.Number:
			c.End = a.End
		default:
			c.End = b.End
		}
	}
	return
}

// Returns true if both ranges are of the same type and
// have identical start/end positions
func (a *EpisodeRange) Equals(b *EpisodeRange) bool {
	if a == b { // pointers to the same thing
		return true
	}
	if a == nil || b == nil {
		return false
	}

	if a.Type == b.Type {
		if a.End == b.End || (a.End != nil && b.End != nil && a.End.Number == b.End.Number) {
			if a.Start == b.Start || a.Start.Number == b.Start.Number {
				return true
			}
		}
	}
	return false
}

func (a *EpisodeRange) touches(b *EpisodeRange) bool {
	if a == nil || b == nil || a.Type != b.Type {
		return false
	}

	switch {
	case a.End == nil:
		switch {
		case b.End == nil:
			// both infinite
			return true

		case b.End.Number >= a.Start.Number-1:
			// {b  [ }  a ...
			// start-1 so it's still true when they're only adjacent
			return true
		}

	case b.End == nil:
		switch {
		case a.End.Number >= b.Start.Number-1:
			// [a  { ]  b ...
			return true
		}

	case a.Start.Number == b.Start.Number || a.End.Number == b.End.Number:
		// touching
		return true

	case a.End.Number < b.End.Number:
		switch {
		case a.End.Number >= b.Start.Number-1:
			// [a  { ]  b}
			return true
		}

	case b.End.Number < a.End.Number:
		switch {
		case b.End.Number >= a.Start.Number-1:
			// {b  [ }  a]
			return true
		}
	}
	return false
}

// Parses a string in the AniDB API range format and converts into an EpisodeRange.
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
