package titles

// Returns true if the given *Anime should be included in the final ResultSet
type ResultFilter func(*Anime) bool

// Returns true if the Anime with the given title should be included in the final ResultSet
type TitleComparer func(string) bool

// Filters a ResultSet according to the given TitleComparer; returns the filtered ResultSet
func (rs ResultSet) FilterByTitles(cmp TitleComparer) ResultSet {
	return rs.Filter(
		func(a *Anime) bool {
			if cmp(a.PrimaryTitle) {
				return true
			}

			for _, m := range []map[string][]Name{
				a.OfficialNames, a.ShortNames, a.Synonyms,
			} {
				for _, names := range m {
					for _, name := range names {
						if cmp(name.Title) {
							return true
						}
					}
				}
			}
			return false
		})
}

// Filters a ResultSet according to the given ResultFilter; returns the filtered ResultSet
func (rs ResultSet) Filter(filter ResultFilter) ResultSet {
	ret := ResultSet{}
	for _, a := range rs {
		if filter(&a) {
			ret[a.AID] = a
		}
	}

	return ret
}
