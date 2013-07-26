package anidb

import (
	"encoding/json"
	"github.com/Kovensky/go-anidb/misc"
	"strconv"
	"time"
)

type MyListAnime struct {
	AID AID

	EpisodesWithState MyListStateMap

	WatchedEpisodes misc.EpisodeList

	EpisodesPerGroup GroupEpisodes

	Cached time.Time
}

type GroupEpisodes map[GID]misc.EpisodeList

func (ge GroupEpisodes) MarshalJSON() ([]byte, error) {
	generic := make(map[string]misc.EpisodeList, len(ge))
	for k, v := range ge {
		generic[strconv.Itoa(int(k))] = v
	}
	return json.Marshal(generic)
}

func (ge GroupEpisodes) UnmarshalJSON(b []byte) error {
	var generic map[string]misc.EpisodeList
	if err := json.Unmarshal(b, &generic); err != nil {
		return err
	}
	for k, v := range generic {
		ik, err := strconv.ParseInt(k, 10, 32)
		if err != nil {
			return err
		}

		ge[GID(ik)] = v
	}

	return nil
}

type MyListStateMap map[MyListState]misc.EpisodeList

func (sm MyListStateMap) MarshalJSON() ([]byte, error) {
	generic := make(map[string]misc.EpisodeList, len(sm))
	for k, v := range sm {
		generic[strconv.Itoa(int(k))] = v
	}
	return json.Marshal(generic)
}

func (sm MyListStateMap) UnmarshalJSON(b []byte) error {
	var generic map[string]misc.EpisodeList
	if err := json.Unmarshal(b, &generic); err != nil {
		return err
	}
	for k, v := range generic {
		ik, err := strconv.ParseInt(k, 10, 32)
		if err != nil {
			return err
		}

		sm[MyListState(ik)] = v
	}

	return nil
}
