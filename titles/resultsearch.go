package titles

// import "sync"

type ResultFilter func(*Anime) bool

type TitleComparer func(string) bool

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

func (rs ResultSet) Filter(filter ResultFilter) ResultSet {
	ret := ResultSet{}
	for _, a := range rs {
		if filter(&a) {
			ret[a.AID] = a
		}
	}

	return ret
}
