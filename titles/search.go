package titles

import (
	"regexp"
	"strings"
)

func (db *TitlesDatabase) ExactSearchN(s string, n int) (matches SearchMatches) {
	return db.doSearchN(func(k string) bool { return s == k }, n)
}

func (db *TitlesDatabase) ExactSearchAll(s string) (matches SearchMatches) {
	return db.ExactSearchN(s, -1)
}

func (db *TitlesDatabase) ExactSearch(s string) (m SearchMatch) {
	return firstMatch(db.ExactSearchN(s, 1))
}

func (db *TitlesDatabase) ExactSearchFoldN(s string, n int) (matches SearchMatches) {
	return db.doSearchN(func(k string) bool { return strings.EqualFold(k, s) }, n)
}

func (db *TitlesDatabase) ExactSearchFoldAll(s string) (matches SearchMatches) {
	return db.ExactSearchFoldN(s, -1)
}

func (db *TitlesDatabase) ExactSearchFold(s string) (m SearchMatch) {
	return firstMatch(db.ExactSearchFoldN(s, 1))
}

func (db *TitlesDatabase) RegexpSearchN(re *regexp.Regexp, n int) (matches SearchMatches) {
	return db.doSearchN(func(k string) bool { return re.MatchString(k) }, n)
}

func (db *TitlesDatabase) RegexpSearchAll(re *regexp.Regexp) (matches SearchMatches) {
	return db.RegexpSearchN(re, -1)
}

func (db *TitlesDatabase) RegexpSearch(re *regexp.Regexp) (m SearchMatch) {
	return firstMatch(db.RegexpSearchN(re, 1))
}

func (db *TitlesDatabase) PrefixSearchN(s string, n int) (matches SearchMatches) {
	return db.doSearchN(func(k string) bool { return strings.HasPrefix(k, s) }, n)
}

func (db *TitlesDatabase) PrefixSearchAll(s string) (matches SearchMatches) {
	return db.PrefixSearchN(s, -1)
}

func (db *TitlesDatabase) PrefixSearch(s string) (m SearchMatch) {
	return firstMatch(db.PrefixSearchN(s, 1))
}

func (db *TitlesDatabase) SuffixSearchN(s string, n int) (matches SearchMatches) {
	return db.doSearchN(func(k string) bool { return strings.HasSuffix(k, s) }, n)
}

func (db *TitlesDatabase) SuffixSearchAll(s string) (matches SearchMatches) {
	return db.SuffixSearchN(s, -1)
}

func (db *TitlesDatabase) SuffixSearch(s string) (m SearchMatch) {
	return firstMatch(db.SuffixSearchN(s, 1))
}

func (db *TitlesDatabase) PrefixSearchFoldN(s string, n int) (matches SearchMatches) {
	s = strings.ToLower(s)
	return db.doSearchN(func(k string) bool { return strings.HasPrefix(strings.ToLower(k), s) }, n)
}

func (db *TitlesDatabase) PrefixSearchFoldAll(s string) (matches SearchMatches) {
	return db.PrefixSearchFoldN(s, -1)
}

func (db *TitlesDatabase) PrefixSearchFold(s string) (m SearchMatch) {
	return firstMatch(db.PrefixSearchFoldN(s, 1))
}

func (db *TitlesDatabase) SuffixSearchFoldN(s string, n int) (matches SearchMatches) {
	s = strings.ToLower(s)
	return db.doSearchN(func(k string) bool { return strings.HasSuffix(strings.ToLower(k), s) }, n)
}

func (db *TitlesDatabase) SuffixSearchFoldAll(s string) (matches SearchMatches) {
	return db.SuffixSearchFoldN(s, -1)
}

func (db *TitlesDatabase) SuffixSearchFold(s string) (m SearchMatch) {
	return firstMatch(db.SuffixSearchFoldN(s, 1))
}

// \b doesn't consider the boundary between e.g. '.' and ' ' in ". "
// to be a word boundary, but . may be significant in a title
const wordBound = ` `

func (db *TitlesDatabase) FuzzySearch(s string) (rs ResultSet) {
	// whole title
	if matches := db.ExactSearchAll(s); len(matches) > 0 {
		// log.Printf("best case: %q", s)
		return matches.ToResultSet(db)
	}

	// all regexes are guaranteed to compile:
	// the user-supplied token already went through regexp.QuoteMeta
	// all other tokens are hardcoded, so a compilation failure is reason for panic

	words := strings.Fields(regexp.QuoteMeta(s))
	q := strings.Join(words, `.*`)

	candidates := db.RegexpSearchAll(regexp.MustCompile(q)).ToResultSet(db)
	if len(candidates) == 0 {
		// log.Printf("no results: %q", s)
		return nil
	}
	q = strings.Join(words, ` `)

	// initial words (prefix, but ending at word boundary)
	re := regexp.MustCompile(`\A` + q + wordBound)
	reCmp := func(k string) bool { return re.MatchString(k) }
	if rs = candidates.FilterByTitles(reCmp); len(rs) > 0 {
		// log.Printf("1st case: %q", s)
		return
	}

	// final words (suffix, but starting at a word boundary)
	re = regexp.MustCompile(wordBound + q + `\z`)
	if rs = candidates.FilterByTitles(reCmp); len(rs) > 0 {
		// log.Printf("2nd case: %q", s)
		return
	}

	// infix words
	re = regexp.MustCompile(wordBound + q + wordBound)
	if rs = candidates.FilterByTitles(reCmp); len(rs) > 0 {
		// log.Printf("3rd case: %q", s)
		return
	}

	// initial substring
	if rs = candidates.FilterByTitles(
		func(k string) bool {
			return strings.HasPrefix(k, s)
		}); len(rs) > 0 {
		// log.Printf("4th case: %q", s)
		return
	}

	// terminal substring
	if rs = candidates.FilterByTitles(
		func(k string) bool {
			return strings.HasSuffix(k, s)
		}); len(rs) > 0 {
		// log.Printf("5th case: %q", s)
		return
	}

	// words in that order, but with possible words between them...
	q = strings.Join(words, ` +(?:[^ ]+ +)*`)

	// ... initial ...
	re = regexp.MustCompile(`\A` + q + wordBound)
	if rs = candidates.FilterByTitles(reCmp); len(rs) > 0 {
		// log.Printf("6th case: %q", s)
		return
	}

	// ... then final ...
	re = regexp.MustCompile(wordBound + q + `\z`)
	if rs = candidates.FilterByTitles(reCmp); len(rs) > 0 {
		// log.Printf("7th case: %q", s)
		return
	}

	// ... then anywhere
	re = regexp.MustCompile(wordBound + q + wordBound)
	if rs = candidates.FilterByTitles(reCmp); len(rs) > 0 {
		// log.Printf("8th case: %q", s)
		return
	}

	// then it's that, but with any or no characters between the input words...
	q = strings.Join(words, `.*`)

	// and the same priority order as for the substring case
	// initial
	re = regexp.MustCompile(`\A` + q)
	if rs = candidates.FilterByTitles(reCmp); len(rs) > 0 {
		// log.Printf("9th case: %q", s)
		return
	}

	// final
	re = regexp.MustCompile(q + `\z`)
	if rs = candidates.FilterByTitles(reCmp); len(rs) > 0 {
		// log.Printf("10th case: %q", s)
		return
	}

	// no result better than the inital candidates
	// log.Printf("worst case: %q", s)
	return candidates
}

func (db *TitlesDatabase) FuzzySearchFold(s string) (rs ResultSet) {
	// whole title
	if matches := db.ExactSearchFoldAll(s); len(matches) > 0 {
		return matches.ToResultSet(db)
	}

	words := strings.Fields(`(?i:` + regexp.QuoteMeta(s) + `)`)
	q := strings.Join(words, `.*`)

	candidates := db.RegexpSearchAll(regexp.MustCompile(q)).ToResultSet(db)
	if len(candidates) == 0 {
		// log.Printf("no results: %q", s)
		return nil
	}
	q = strings.Join(words, `\s+`)

	// initial words (prefix, but ending at word boundary)
	re := regexp.MustCompile(`\A` + q + wordBound)
	reCmp := func(k string) bool { return re.MatchString(k) }
	if rs = candidates.FilterByTitles(reCmp); len(rs) > 0 {
		// log.Printf("1st case: %q", s)
		return
	}

	// final words (suffix, but starting at a word boundary)
	re = regexp.MustCompile(wordBound + q + `\z`)
	if rs = candidates.FilterByTitles(reCmp); len(rs) > 0 {
		// log.Printf("2nd case: %q", s)
		return
	}

	// infix words
	re = regexp.MustCompile(wordBound + q + wordBound)
	if rs = candidates.FilterByTitles(reCmp); len(rs) > 0 {
		// log.Printf("3rd case: %q", s)
		return
	}

	// initial substring
	ls := strings.ToLower(s)
	if rs = candidates.FilterByTitles(
		func(k string) bool {
			return strings.HasPrefix(strings.ToLower(k), ls)
		}); len(rs) > 0 {
		// log.Printf("4th case: %q", s)
		return
	}

	// terminal substring
	if rs = candidates.FilterByTitles(
		func(k string) bool {
			return strings.HasSuffix(strings.ToLower(k), ls)
		}); len(rs) > 0 {
		// log.Printf("5th case: %q", s)
		return
	}

	// words in that order, but with possible words between them...
	q = strings.Join(words, `\s+(?:\S+\s+)*`)

	// ... initial ...
	re = regexp.MustCompile(`\A` + q + wordBound)
	if rs = candidates.FilterByTitles(reCmp); len(rs) > 0 {
		// log.Printf("6th case: %q", s)
		return
	}

	// ... then final ...
	re = regexp.MustCompile(wordBound + q + `\z`)
	if rs = candidates.FilterByTitles(reCmp); len(rs) > 0 {
		// log.Printf("7th case: %q", s)
		return
	}

	// ... then anywhere
	re = regexp.MustCompile(wordBound + q + wordBound)
	if rs = candidates.FilterByTitles(reCmp); len(rs) > 0 {
		// log.Printf("8th case: %q", s)
		return
	}

	// then it's that, but with any or no characters between the input words...
	q = strings.Join(words, `.*`)

	// and the same priority order as for the substring case
	// initial
	re = regexp.MustCompile(`\A` + q)
	if rs = candidates.FilterByTitles(reCmp); len(rs) > 0 {
		// log.Printf("9th case: %q", s)
		return
	}

	// final
	re = regexp.MustCompile(q + `\z`)
	if rs = candidates.FilterByTitles(reCmp); len(rs) > 0 {
		// log.Printf("10th case: %q", s)
		return
	}

	// no result better than the inital candidates
	// log.Printf("worst case: %q", s)
	return candidates
}
