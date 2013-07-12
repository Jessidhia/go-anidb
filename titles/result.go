package titles

import (
	"sort"
)

type ResultSet map[AID]Anime // Mapping of AIDs to Anime
type Results []Anime

// Returns a slice with the AIDs of all anime in the Results.
func (res Results) AIDList() (aid []AID) {
	aid = make([]AID, 0, len(res))
	for _, r := range res {
		aid = append(aid, r.AID)
	}
	return
}

// Converts the SearchMatches (which usually contains various duplicates)
// into a ResultSet. Needs the same TitlesDatabase as was used to generate the
// SearchMatches.
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

// Returns true if the first parameter should sort before the second parameter
type ResultComparer func(*Anime, *Anime) bool

var (
	aidSort = func(a *Anime, b *Anime) bool {
		return a.AID < b.AID
	}
	titleSort = func(a *Anime, b *Anime) bool {
		return sort.StringSlice{a.PrimaryTitle, b.PrimaryTitle}.Less(0, 1)
	}
)

// Returns the results sorted by AID
func (rs ResultSet) ResultsByAID() (res Results) {
	return rs.ResultsByFunc(aidSort)
}

// Returns the results in inverse AID sort
func (rs ResultSet) ReverseResultsByAID() (res Results) {
	return rs.ReverseResultsByFunc(aidSort)
}

// Returns the results sorted by Primary Title
func (rs ResultSet) ResultsByPrimaryTitle() (res Results) {
	return rs.ResultsByFunc(titleSort)
}

// Returns the results in inverse Primary Title sort
func (rs ResultSet) ReverseResultsByPrimaryTitle() (res Results) {
	return rs.ReverseResultsByFunc(titleSort)
}

// Returns the results sorted according to the given ResultComparer
func (rs ResultSet) ResultsByFunc(f ResultComparer) (res Results) {
	res = rs.unsortedResults()
	f.Sort(res)
	return
}

// Returns the results sorted inversely according to the given ResultComparer
func (rs ResultSet) ReverseResultsByFunc(f ResultComparer) (res Results) {
	res = rs.unsortedResults()
	f.ReverseSort(res)
	return
}
