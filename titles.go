package anidb

import (
	"github.com/Kovensky/go-anidb/titles"
)

// Searches for the given anime name, case sensitive.
//
// Returns the match with the smallest AID.
func SearchAnime(name string) AID {
	rs := SearchAnimeAll(name).ResultsByAID()
	if len(rs) == 0 {
		return 0
	}
	return AID(rs[0].AID)
}

// Searches for all anime that match the given anime name, case sensitive.
func SearchAnimeAll(name string) titles.ResultSet {
	if name == "" {
		return nil
	}
	return titlesDB.FuzzySearch(name)
}

// Searches for the given anime name, case folding.
//
// Returns the match with the smallest AID.
func SearchAnimeFold(name string) AID {
	rs := SearchAnimeFoldAll(name).ResultsByAID()
	if len(rs) == 0 {
		return 0
	}
	return AID(rs[0].AID)
}

// Searches for all anime that match the given anime name, case folding.
func SearchAnimeFoldAll(name string) titles.ResultSet {
	if name == "" {
		return nil
	}
	return titlesDB.FuzzySearchFold(name)
}
