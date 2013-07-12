package titles

import (
	"sync"
)

type SearchMatch struct {
	Matched string
	AID     int
}

type SearchMatches []SearchMatch

func searchFunc(wg *sync.WaitGroup, ret chan SearchMatch, t *TitleMap, cmp func(string) bool) {
	defer wg.Done()

	for _, m := range []map[string]int{t.ShortMap, t.OfficialMap, t.SynonymMap} {
		for k, v := range m {
			if cmp(k) {
				ret <- SearchMatch{Matched: k, AID: v}
			}
		}
	}
}

func (db *TitlesDatabase) multiSearch(cmp func(string) bool) (matches chan SearchMatch) {
	db.RLock()

	matches = make(chan SearchMatch, 100)

	go func() {
		defer db.RUnlock()

		match := make(chan SearchMatch, 100)

		for k, v := range db.PrimaryMap {
			if cmp(k) {
				matches <- SearchMatch{Matched: k, AID: v}
			}
		}

		wg := &sync.WaitGroup{}

		for _, a := range db.LanguageMap {
			wg.Add(1)
			go searchFunc(wg, match, a, cmp)
		}
		go func() { wg.Wait(); close(match) }()

		for m := range match {
			matches <- m
		}
		close(matches)
	}()
	return matches
}

func (db *TitlesDatabase) doSearchN(cmp func(string) bool, n int) (matches SearchMatches) {
	if n == 0 {
		return nil
	}

	ch := db.multiSearch(cmp)
	if n > 0 {
		matches = make(SearchMatches, 0, n)
		for m := range ch {
			matches = append(matches, m)
			if len(matches) == n {
				go func() {
					for _ = range ch {
						// drain channel
					}
				}()
				return matches[:n]
			}
		}
	} else {
		for m := range ch {
			matches = append(matches, m)
		}
	}
	return
}

func firstMatch(matches SearchMatches) (m SearchMatch) {
	if len(matches) > 0 {
		m = matches[0]
	}
	return
}

func (db *TitlesDatabase) doSearch1(cmp func(string) bool) (m SearchMatch) {
	return firstMatch(db.doSearchN(cmp, 1))
}
