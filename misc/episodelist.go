package misc

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type EpisodeCount struct {
	RegularCount int
	SpecialCount int
	CreditsCount int
	OtherCount   int
	TrailerCount int
	ParodyCount  int
}

type EpisodeList []*EpisodeRange

func EpisodeToList(ep *Episode) EpisodeList {
	return RangesToList(EpisodeToRange(ep))
}

func RangesToList(ranges ...*EpisodeRange) EpisodeList {
	return EpisodeList(ranges)
}

func ContainerToList(ec EpisodeContainer) EpisodeList {
	switch v := ec.(type) {
	case *Episode:
		return EpisodeToList(v)
	case *EpisodeRange:
		return RangesToList(v)
	case EpisodeList:
		return v
	default:
		panic("unimplemented")
	}
}

// Converts the EpisodeList into the AniDB API list format.
func (el EpisodeList) String() string {
	scales := map[EpisodeType]int{}

	for _, er := range el {
		if er == nil {
			continue
		}

		s := er.scale()
		if s > scales[er.Type] {
			scales[er.Type] = s
		}
	}

	parts := make([]string, len(el))
	for i, er := range el {
		parts[i] = er.Format(scales[er.Type])
	}

	return strings.Join(parts, ",")
}

// Formats the list according to the number of digits of
// the count for its type, given in the EpisodeCount.
func (el EpisodeList) FormatLog(ec EpisodeCount) string {
	parts := make([]string, len(el))
	for i, er := range el {
		switch er.Type {
		case EpisodeTypeRegular:
			parts[i] = er.FormatLog(ec.RegularCount)
		case EpisodeTypeSpecial:
			parts[i] = er.FormatLog(ec.SpecialCount)
		case EpisodeTypeCredits:
			parts[i] = er.FormatLog(ec.CreditsCount)
		case EpisodeTypeOther:
			parts[i] = er.FormatLog(ec.OtherCount)
		case EpisodeTypeTrailer:
			parts[i] = er.FormatLog(ec.TrailerCount)
		case EpisodeTypeParody:
			parts[i] = er.FormatLog(ec.ParodyCount)
		default:
			parts[i] = er.Format(er.scale())
		}
	}

	return strings.Join(parts, ",")
}

func (el EpisodeList) Infinite() bool {
	for i := range el {
		if el[i].Infinite() {
			return true
		}
	}
	return false
}

// Returns a channel that can be used to iterate using for/range.
//
// If the EpisodeList is infinite, then the channel is also infinite.
// The caller is allowed to close the channel in such case.
//
// NOTE: Not thread safe.
func (el EpisodeList) Episodes() chan Episode {
	ch := make(chan Episode, 1)

	go func() {
		abort := false

		if el.Infinite() {
			defer func() { recover(); abort = true }()
		} else {
			defer close(ch)
		}

		for _, er := range el {
			for ep := range er.Episodes() {
				ch <- ep

				if abort {
					return
				}
			}
		}
	}()
	return ch
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

	return el.Simplify()
}

// Returns a simplified version of the EpisodeList (removes nil ranges, merges mergeable ranges, sorts).
func (el EpisodeList) Simplify() EpisodeList {
	nl := make(EpisodeList, 0, len(el))

	// drop nil ranges
	for _, er := range el {
		if er != nil {
			nl = append(nl, er)
		}
	}

	// merge ranges
	for n, changed := 0, true; changed; n++ {
		tmp := EpisodeList{}
		used := map[int]bool{}
		changed = false

		for i, a := range nl {
			if used[i] {
				continue
			}
			for j, b := range nl[i+1:] {
				if c := a.Merge(b); c != nil {
					changed = true
					used[j+i+1] = true
					a = c
				}
			}
			tmp = append(tmp, a)
		}
		nl = tmp

		if n > len(el) {
			panic(fmt.Sprintf("Too many iterations (%d) when simplifing %s!", n, el))
		}
	}
	sort.Sort(nl)
	return nl
}

func (el EpisodeList) CountEpisodes() (ec EpisodeCount) {
	for _, er := range el {
		var c *int
		switch er.Type {
		case EpisodeTypeRegular:
			c = &ec.RegularCount
		case EpisodeTypeSpecial:
			c = &ec.SpecialCount
		case EpisodeTypeCredits:
			c = &ec.CreditsCount
		case EpisodeTypeOther:
			c = &ec.OtherCount
		case EpisodeTypeTrailer:
			c = &ec.TrailerCount
		case EpisodeTypeParody:
			c = &ec.ParodyCount
		default:
			continue
		}
		if *c < 0 {
			continue
		}
		if er.End == nil {
			*c = -1
			continue
		}
		*c += er.End.Number - er.Start.Number
	}
	return
}

func (el EpisodeList) Len() int {
	return len(el)
}

func (el EpisodeList) Less(i, j int) bool {
	switch {
	case el[i] == nil:
		return true
	case el[j] == nil:
		return false
	case el[i].Type < el[j].Type:
		return true
	case el[i].Type > el[j].Type:
		return false
	case el[i].Start.Number < el[j].Start.Number:
		return true
	}
	return false
}

func (el EpisodeList) Swap(i, j int) {
	el[i], el[j] = el[j], el[i]
}

func (el *EpisodeList) Add(ec EpisodeContainer) {
	*el = append(*el, ContainerToList(ec)...)
	*el = el.Simplify()
}

func (el *EpisodeList) Sub(ec EpisodeContainer) {
	el2 := make(EpisodeList, 0, len(*el)*2)
	switch e, ok := ec.(canInfinite); {
	case ok:
		if e.Infinite() {
			eCh := e.Episodes()
			ep := <-eCh
			close(eCh)

			for _, r := range *el {
				el2 = append(el2, r.Split(&ep)[0])
			}
			break
		}
		fallthrough
	default:
		for ep := range ec.Episodes() {
			for _, r := range *el {
				el2 = append(el2, r.Split(&ep)...)
			}
			el2 = el2.Simplify()
		}
	}
	*el = append(*el, el2.Simplify()...)
}

// Equivalent to marshaling el.String()
func (el EpisodeList) MarshalJSON() ([]byte, error) {
	return json.Marshal(el.String())
}

// NOTE: Since the String() representation doesn't include them,
// it's not exactly reversible if the user has set .Parts in any
// of the contained episodes.
func (el EpisodeList) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	l := ParseEpisodeList(v)
	for k := range l {
		el[k] = l[k]
	}
	return nil
}
