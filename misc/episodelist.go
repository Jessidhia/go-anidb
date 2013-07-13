package misc

import (
	"fmt"
	"sort"
	"strings"
)

type EpisodeList []*EpisodeRange

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
