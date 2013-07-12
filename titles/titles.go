package titles

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"io/ioutil"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var _ = json.Decoder{}

const (
	DataDumpURL = "http://anidb.net/api/anime-titles.dat.gz"
)

type Name struct {
	Language string
	Title    string
}

type Anime struct {
	AID          int
	PrimaryTitle string

	OfficialNames map[string][]Name
	Synonyms      map[string][]Name
	ShortNames    map[string][]Name
}

type TitleMap struct {
	Language string

	OfficialMap map[string]int
	SynonymMap  map[string]int
	ShortMap    map[string]int
}

type TitlesDatabase struct {
	sync.RWMutex
	UpdateTime time.Time
	Languages  []string

	LanguageMap map[string]*TitleMap
	PrimaryMap  map[string]int

	AnimeMap map[int]*Anime
}

var createdRegexp = regexp.MustCompile(`^# created: (.*)$`)

func (db *TitlesDatabase) LoadDB(r io.ReadCloser) {
	db.Lock()
	defer db.Unlock()

	all, _ := ioutil.ReadAll(r)
	r.Close()

	var rd io.Reader
	if gz, err := gzip.NewReader(bytes.NewReader(all)); err == nil {
		defer gz.Close()
		rd = gz
	} else {
		rd = bytes.NewReader(all)
	}
	sc := bufio.NewScanner(rd)

	if db.PrimaryMap == nil {
		db.PrimaryMap = map[string]int{}
	}
	if db.LanguageMap == nil {
		db.LanguageMap = map[string]*TitleMap{}
	}
	if db.AnimeMap == nil {
		db.AnimeMap = map[int]*Anime{}
	}

	allLangs := map[string]struct{}{}
	for sc.Scan() {
		s := sc.Text()

		if s[0] == '#' {
			cr := createdRegexp.FindStringSubmatch(s)

			if len(cr) > 1 && cr[1] != "" {
				db.UpdateTime, _ = time.Parse(time.ANSIC, cr[1])
			}
			continue
		}

		parts := strings.Split(s, "|")
		if len(parts) < 4 {
			continue
		}

		aid, _ := strconv.ParseInt(parts[0], 10, 32)
		typ, _ := strconv.ParseInt(parts[1], 10, 8)

		if _, ok := db.AnimeMap[int(aid)]; !ok {
			db.AnimeMap[int(aid)] = &Anime{
				AID:           int(aid),
				OfficialNames: map[string][]Name{},
				Synonyms:      map[string][]Name{},
				ShortNames:    map[string][]Name{},
			}
		}

		lang, title := parts[2], parts[3]
		allLangs[lang] = struct{}{}

		switch typ {
		case 1: // primary
			db.PrimaryMap[title] = int(aid)

			db.AnimeMap[int(aid)].PrimaryTitle = strings.Replace(title, "`", "'", -1)
		case 2: // synonym
			lm, ok := db.LanguageMap[lang]
			if !ok {
				lm = db.makeLangMap(lang)
			}
			lm.SynonymMap[title] = int(aid)

			db.AnimeMap[int(aid)].Synonyms[lang] = append(db.AnimeMap[int(aid)].Synonyms[lang],
				Name{Language: lang, Title: strings.Replace(title, "`", "'", -1)})
		case 3: // short
			lm, ok := db.LanguageMap[lang]
			if !ok {
				lm = db.makeLangMap(lang)
			}
			lm.ShortMap[title] = int(aid)

			db.AnimeMap[int(aid)].ShortNames[lang] = append(db.AnimeMap[int(aid)].Synonyms[lang],
				Name{Language: lang, Title: strings.Replace(title, "`", "'", -1)})
		case 4: // official
			lm, ok := db.LanguageMap[lang]
			if !ok {
				lm = db.makeLangMap(lang)
			}
			lm.OfficialMap[title] = int(aid)

			db.AnimeMap[int(aid)].OfficialNames[lang] = append(db.AnimeMap[int(aid)].Synonyms[lang],
				Name{Language: lang, Title: strings.Replace(title, "`", "'", -1)})
		}
	}
	langs := make([]string, 0, len(allLangs))
	for k, _ := range allLangs {
		langs = append(langs, k)
	}
	sort.Strings(langs)
	db.Languages = langs
}

func (db *TitlesDatabase) makeLangMap(lang string) *TitleMap {
	tm := &TitleMap{
		Language:    lang,
		OfficialMap: map[string]int{},
		SynonymMap:  map[string]int{},
		ShortMap:    map[string]int{},
	}
	db.LanguageMap[lang] = tm
	return tm
}
