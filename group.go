package anidb

import (
	"encoding/json"
	"strconv"
	"time"
)

// See the constants list for valid values.
type GroupRelationType int

const (
	GroupParticipantIn = GroupRelationType(1 + iota)
	GroupParentOf
	_
	GroupMergedFrom
	GroupNowKnownAs
	GroupOther

	GroupChildOf = GroupRelationType(102)
)

func (gr GroupRelationType) String() string {
	switch gr {
	case GroupParticipantIn:
		return "Participated In"
	case GroupParentOf:
		return "Parent Of"
	case GroupMergedFrom:
		return "Merged From"
	case GroupNowKnownAs:
		return "Now Known As"
	case GroupOther:
		return "Other"
	case GroupChildOf:
		return "Child Of"
	default:
		return "Unknown"
	}
}

type GroupRelations map[GID]GroupRelationType

func (gr GroupRelations) MarshalJSON() ([]byte, error) {
	generic := make(map[string]int, len(gr))
	for k, v := range gr {
		generic[strconv.Itoa(int(k))] = int(v)
	}
	return json.Marshal(generic)
}

func (gr GroupRelations) UnmarshalJSON(b []byte) error {
	var generic map[string]int
	if err := json.Unmarshal(b, &generic); err != nil {
		return err
	}
	for k, v := range generic {
		ik, err := strconv.ParseInt(k, 10, 32)
		if err != nil {
			return err
		}

		gr[GID(ik)] = GroupRelationType(v)
	}

	return nil
}

type Group struct {
	GID GID

	Name      string // Full name
	ShortName string // Abbreviated name

	IRC     string // irc: schema format
	URL     string
	Picture string

	Founded   time.Time
	Disbanded time.Time

	LastRelease  time.Time
	LastActivity time.Time

	Rating     Rating
	AnimeCount int // Number of anime this group has worked on
	FileCount  int // Number of files this group has released

	RelatedGroups GroupRelations

	Cached time.Time
}
