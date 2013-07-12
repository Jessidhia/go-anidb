package titles

import (
	"sort"
)

type ResultSet map[int]Anime
type Results []Anime

func (res Results) AIDList() (aid []int) {
	aid = make([]int, 0, len(res))
	for _, r := range res {
		aid = append(aid, r.AID)
	}
	return
}

func (matches SearchMatches) ToResultSet(db *TitlesDatabase) (rs ResultSet) {
	if matches == nil {
		return nil
	}
	db.RLock()
	defer db.RUnlock()

	rs = ResultSet{}
	for _, m := range matches {
		rs[m.AID] = *db.AnimeMap[m.AID]
	}
	return
}

func (rs ResultSet) unsortedResults() (res Results) {
	res = make(Results, 0, len(rs))
	for _, r := range rs {
		res = append(res, r)
	}
	return
}

// Returns true if the first parameter is less than the second parameter
type ResultComparer func(*Anime, *Anime) bool

var (
	aidSort = func(a *Anime, b *Anime) bool {
		return a.AID < b.AID
	}
	titleSort = func(a *Anime, b *Anime) bool {
		return sort.StringSlice{a.PrimaryTitle, b.PrimaryTitle}.Less(0, 1)
	}
)

func (rs ResultSet) ResultsByAID() (res Results) {
	return rs.ResultsByFunc(aidSort)
}

func (rs ResultSet) ReverseResultsByAID() (res Results) {
	return rs.ReverseResultsByFunc(aidSort)
}

func (rs ResultSet) ResultsByPrimaryTitle() (res Results) {
	return rs.ResultsByFunc(titleSort)
}

func (rs ResultSet) ReverseResultsByPrimaryTitle() (res Results) {
	return rs.ReverseResultsByFunc(titleSort)
}

func (rs ResultSet) ResultsByFunc(f ResultComparer) (res Results) {
	res = rs.unsortedResults()
	f.Sort(res)
	return
}

func (rs ResultSet) ReverseResultsByFunc(f ResultComparer) (res Results) {
	res = rs.unsortedResults()
	f.Sort(res)
	return
}
