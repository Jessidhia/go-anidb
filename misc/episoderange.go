package misc

import (
	"fmt"
	"strings"
)

type canInfinite interface {
	EpisodeContainer
	Infinite() bool
}

// A range of episodes with a start and possibly without an end.
type EpisodeRange struct {
	Type  EpisodeType // Must be equal to both the Start and End types; if End is nil, must be equal to the Start type
	Start *Episode    // The start of the range
	End   *Episode    // The end of the range; may be nil, which represents an endless range
}

func EpisodeToRange(ep *Episode) *EpisodeRange {
	return &EpisodeRange{
		Type:  ep.Type,
		Start: ep,
		End:   ep,
	}
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

func (er *EpisodeRange) Infinite() bool {
	return er != nil && er.End == nil
}

// Returns the number of episodes that Episodes() would return.
//
// Returns -1 for infinite ranges.
func (er *EpisodeRange) Len() int {
	c := 0
	switch {
	case er.Infinite():
		c = -1
	case er.Valid():
		if er.Start.Parts > 0 {
			c += er.Start.Parts - er.Start.Part
		}
		c += 1 + er.End.Number - er.Start.Number
		if er.End.Part > 0 {
			c += er.End.Part
		}
	}
	return c
}

// Returns a channel that can be used to iterate using for/range.
//
// If the EpisodeRange is infinite, then the channel is also infinite.
// The caller is allowed to close the channel in such case.
func (er *EpisodeRange) Episodes() chan Episode {
	ch := make(chan Episode, 1)
	if !er.Valid() {
		close(ch)
		return ch
	}

	start := *er.Start
	inf := er.Infinite()
	end := Episode{}
	if !inf {
		end = *er.End
	}

	go func() {
		abort := false

		if inf {
			// we allow the caller to close the channel on infinite lists
			defer func() { recover(); abort = true }()
		} else {
			defer close(ch)
		}

		ep := start

		switch {
		case inf:
			for ; !abort && ep.Parts > 0 && ep.Number == start.Number; ep.IncPart() {
				ch <- ep
			}
			for ; !abort; ep.IncNumber() {
				ch <- ep
			}
		case start.Part == -1 && end.Part == -1:
			for ; ep.Number <= end.Number; ep.IncNumber() {
				ch <- ep
			}
		case start.Parts > 0:
			for ; ep.Number == start.Number; ep.IncPart() {
				ch <- ep
			}
			fallthrough
		default:
			for ; ep.Number < end.Number; ep.IncNumber() {
				ch <- ep
			}
			if end.Part >= 0 {
				ep.Part = 0
			}
			for ; ep.Part <= end.Part; ep.IncPart() {
				ch <- ep
			}
		}
	}()
	return ch
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
	if !er.Valid() {
		return false
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

		switch {
		case a.Start.Number == b.Start.Number:
			switch {
			case a.Start.Part < 0:
				c.Start = a.Start
			case b.Start.Part < 0:
				c.Start = b.Start
			case a.Start.Part <= b.Start.Part:
				c.Start = a.Start
			default:
				c.Start = b.Start
			}
		case a.Start.Number < b.Start.Number:
			c.Start = a.Start
		default:
			c.Start = b.Start
		}

		switch {
		case a.End == nil || b.End == nil:
			c.End = nil
		case a.End.Number == b.End.Number:
			switch {
			case a.End.Part < 0:
				c.End = a.End
			case b.End.Part < 0:
				c.End = b.End
			case a.End.Part >= b.End.Part:
				c.End = a.End
			default:
				c.End = b.End
			}
		case a.End.Number > b.End.Number:
			c.End = a.End
		default:
			c.End = b.End
		}
	}
	return
}

// Check if the given range is not nil, has a defined start
// and, if it has an end, that the end ends after the start.
func (er *EpisodeRange) Valid() bool {
	switch {
	case er == nil, er.Start == nil:
		return false
	case er.End == nil:
		return true
	case er.Start.Number < er.End.Number:
		return true
	case er.Start.Number > er.End.Number:
		return false
	case er.Start.Part <= er.End.Part:
		return true
	default:
		return false
	}
}

// Simplifies the Start/End ranges if one contains the other.
// Sets the pointers to be identical if the range is modified.
//
// Modifies in-place, returns itself.
func (er *EpisodeRange) Simplify() *EpisodeRange {
	switch {
	case er.Start.ContainsEpisodes(er.End):
		er.End = er.Start
	case er.End != nil && er.End.ContainsEpisodes(er.Start):
		er.Start = er.End
	}
	return er
}

// Splits the range into one or two ranges, using the given
// Episode as the split point. The Episode is not included in
// the resulting ranges.
func (er *EpisodeRange) Split(ep *Episode) []*EpisodeRange {
	if !er.Valid() {
		return []*EpisodeRange{nil, nil}
	}
	if !er.ContainsEpisodes(ep) { // implies same type
		return []*EpisodeRange{er}
	}

	a := *er.Start

	inf := er.End == nil
	b := Episode{}
	if !inf {
		b = *er.End
	}

	end := &b
	if inf {
		end = nil
	}

	switch {
	case a.ContainsEpisodes(ep) && b.ContainsEpisodes(ep):
		return []*EpisodeRange{nil, nil}
	case a.ContainsEpisodes(ep):
		if ep.Part >= 0 {
			a.IncPart()
		} else {
			a.IncNumber()
		}
		if a.Number == b.Number && b.Parts > 0 {
			a.Parts = b.Parts
		}

		r := &EpisodeRange{
			Type:  er.Type,
			Start: &a,
			End:   end,
		}
		return []*EpisodeRange{nil, r.Simplify()}
	case b.ContainsEpisodes(ep):
		if ep.Part >= 0 {
			b.DecPart()
		} else {
			b.DecNumber()
		}
		if b.Number == a.Number {
			if a.Parts > 0 {
				b.Parts = a.Parts
				b.Part = a.Parts - 1
			} else if b.Part < 0 {
				b.Part = a.Part
			}
		}
		r := &EpisodeRange{
			Type:  er.Type,
			Start: &a,
			End:   &b,
		}
		return []*EpisodeRange{r.Simplify(), nil}
	default:
		ra := &EpisodeRange{
			Type:  er.Type,
			Start: &a,
			End:   ep,
		}
		rb := &EpisodeRange{
			Type:  er.Type,
			Start: ep,
			End:   end,
		}

		ra = ra.Split(ep)[0]
		rb = rb.Split(ep)[1]

		if ra.Valid() {
			ra.Simplify()
		} else {
			ra = nil
		}
		if rb.Valid() {
			rb.Simplify()
		} else {
			rb = nil
		}

		return []*EpisodeRange{ra, rb}
	}
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
		if a.End == b.End || (a.End != nil && b.End != nil &&
			a.End.Number == b.End.Number && a.End.Part == b.End.Part) {
			if a.Start == b.Start || a.Start.Number == b.Start.Number && a.Start.Part == b.Start.Part {
				return true
			}
		}
	}
	return false
}

// CORNER CASE: e.g. 1.3,2.0 (or 1.3,2) never touch,
// unless it's known that 1.3 is the last part.
func (a *EpisodeRange) touches(b *EpisodeRange) bool {
	if a == nil || b == nil || a.Type != b.Type {
		return false
	}

	switch {
	case a == b:
		// log.Println("same pointers")
		return true
	case a.Start == b.Start, a.End != nil && a.End == b.End:
		// log.Println("share pointers")
		return true

	case a.End == nil:
		switch {
		case b.End == nil:
			// log.Println("both infinite")
			return true

		case b.End.Number == a.Start.Number:
			switch {
			// either is whole, or parts are adjacent/overlap
			case b.End.Part == -1, a.Start.Part == -1,
				b.End.Part >= a.Start.Part-1:
				// log.Printf("{ %s [} %s ...", b.End, a.Start)
				return true
			}
		// only if start of next range is whole or is first part
		case b.End.Number == a.Start.Number-1 && a.Start.Part <= 0:
			switch {
			// end is whole, or is last part, or part count is unknown
			case b.End.Part == -1, b.End.Parts == 0,
				b.End.Part == b.End.Parts:
				// log.Printf("{ %s }[ %s ...", b.End, a.Start)
				return true
			}
		case b.End.Number > a.Start.Number:
			// log.Printf("{ %s [ } %s ...", b.End, a.Start)
			return true
		}

	case b.End == nil:
		switch {
		case a.End.Number == b.Start.Number:
			switch {
			case a.End.Part == -1, b.Start.Part == -1,
				a.End.Part >= b.Start.Part-1:
				// log.Printf("[ %s {] %s ...", a.End, b.Start)
				return true
			}
		case a.End.Number == b.Start.Number-1 && b.Start.Part <= 0:
			switch {
			case a.End.Part == -1, a.End.Parts == 0,
				a.End.Part == a.End.Parts:
				// log.Printf("[ %s ]{ %s ...", a.End, b.Start)
				return true
			}
		case a.End.Number > b.Start.Number:
			// log.Printf("[ %s { ] %s ...", a.End, b.Start)
			return true
		}

	case a.Start.Number == b.Start.Number:
		// touching
		switch {
		// either is whole, or parts are immediately adjacent
		case a.Start.Part == -1, b.Start.Part == -1,
			a.Start.Part == b.Start.Part,
			a.Start.Part == b.Start.Part-1,
			a.Start.Part == b.Start.Part+1:
			// log.Printf("[{ %s - %s ]}", a.End, b.Start)
			return true
		}
	case a.End.Number == b.End.Number:
		switch {
		case a.End.Part == -1, b.End.Part == -1,
			a.End.Part == b.End.Part,
			a.End.Part == b.End.Part-1,
			a.End.Part == b.End.Part+1:
			// log.Printf("{[ %s - %s }]", b.End, a.Start)
			return true
		}

	case a.End.Number < b.End.Number:
		switch {
		case a.End.Number == b.Start.Number:
			switch {
			case a.End.Part == -1, b.Start.Part == -1,
				a.End.Part >= b.Start.Part-1:
				// log.Printf("[ %s {] %s }", a.End, b.Start)
				return true
			}
		case a.End.Number == b.Start.Number-1 && b.Start.Part <= 0:
			switch {
			case a.End.Part == -1, a.End.Part == a.End.Parts-1,
				a.End.Part == b.Start.Parts:
				// log.Printf("[ %s ]{ %s }", a.End, b.Start)
				return true
			}
		case a.End.Number > b.Start.Number:
			// log.Printf("[ %s { ] %s }", a.End, b.Start)
			return true
		}

	case b.End.Number < a.End.Number:
		switch {
		case b.End.Number == a.Start.Number:
			switch {
			case b.End.Part == -1, a.Start.Part == -1,
				b.End.Part >= a.Start.Part-1:
				// log.Printf("{ %s [} %s ]", b.End, a.Start)
				return true
			}
		case b.End.Number == a.Start.Number-1 && a.Start.Part <= 0:
			switch {
			case b.End.Part == -1, b.End.Part == b.End.Parts-1,
				b.End.Part == b.End.Parts:
				// log.Printf("{ %s }[ %s ]", b.End, a.Start)
				return true
			}
		case b.End.Number > a.Start.Number:
			// log.Printf("{ %s [ } %s ]", b.End, a.Start)
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
