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

// Anime ID
type AID int

type Name struct {
	Language string // ISO-ish language string
	Title    string
}

type Anime struct {
	AID          AID
	PrimaryTitle string // The primary title ("x-jat main" title in the HTTP API)

	OfficialNames map[string][]Name
	Synonyms      map[string][]Name
	ShortNames    map[string][]Name
}

// Maps titles in the given language to AIDs
type TitleMap struct {
	Language string // ISO-ish language string

	OfficialMap map[string]AID
	SynonymMap  map[string]AID
	ShortMap    map[string]AID
}

type TitlesDatabase struct {
	sync.RWMutex
	UpdateTime time.Time
	Languages  []string // List of all the languages present in the database

	LanguageMap map[string]*TitleMap // Per-language map (key is ISO-ish language string)
	PrimaryMap  map[string]AID       // Primary title to AID map (language is always "x-jat")

	AnimeMap map[AID]*Anime
}

var createdRegexp = regexp.MustCompile(`^# created: (.*)$`)

// Loads the database from the given io.Reader.
//
// The Reader must point to a file or stream with
// the contents of anime-titles.dat, which can be obtained
// from the DataDumpURL. LoadDB will automatically try to
// un-gzip, so the file can be stored in gzip format.
//
// Note: LoadDB will read the entire contents of the given
// io.Reader.
func (db *TitlesDatabase) LoadDB(r io.Reader) {
	db.Lock()
	defer db.Unlock()

	all, _ := ioutil.ReadAll(r)

	var rd io.Reader
	if gz, err := gzip.NewReader(bytes.NewReader(all)); err == nil {
		defer gz.Close()
		rd = gz
	} else {
		rd = bytes.NewReader(all)
	}
	sc := bufio.NewScanner(rd)

	if db.PrimaryMap == nil {
		db.PrimaryMap = map[string]AID{}
	}
	if db.LanguageMap == nil {
		db.LanguageMap = map[string]*TitleMap{}
	}
	if db.AnimeMap == nil {
		db.AnimeMap = map[AID]*Anime{}
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

		if _, ok := db.AnimeMap[AID(aid)]; !ok {
			db.AnimeMap[AID(aid)] = &Anime{
				AID:           AID(aid),
				OfficialNames: map[string][]Name{},
				Synonyms:      map[string][]Name{},
				ShortNames:    map[string][]Name{},
			}
		}

		lang, title := parts[2], parts[3]
		allLangs[lang] = struct{}{}

		switch typ {
		case 1: // primary
			db.PrimaryMap[title] = AID(aid)

			db.AnimeMap[AID(aid)].PrimaryTitle = strings.Replace(title, "`", "'", -1)
		case 2: // synonym
			lm, ok := db.LanguageMap[lang]
			if !ok {
				lm = db.makeLangMap(lang)
			}
			lm.SynonymMap[title] = AID(aid)

			db.AnimeMap[AID(aid)].Synonyms[lang] = append(db.AnimeMap[AID(aid)].Synonyms[lang],
				Name{Language: lang, Title: strings.Replace(title, "`", "'", -1)})
		case 3: // short
			lm, ok := db.LanguageMap[lang]
			if !ok {
				lm = db.makeLangMap(lang)
			}
			lm.ShortMap[title] = AID(aid)

			db.AnimeMap[AID(aid)].ShortNames[lang] = append(db.AnimeMap[AID(aid)].Synonyms[lang],
				Name{Language: lang, Title: strings.Replace(title, "`", "'", -1)})
		case 4: // official
			lm, ok := db.LanguageMap[lang]
			if !ok {
				lm = db.makeLangMap(lang)
			}
			lm.OfficialMap[title] = AID(aid)

			db.AnimeMap[AID(aid)].OfficialNames[lang] = append(db.AnimeMap[AID(aid)].Synonyms[lang],
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
		OfficialMap: map[string]AID{},
		SynonymMap:  map[string]AID{},
		ShortMap:    map[string]AID{},
	}
	db.LanguageMap[lang] = tm
	return tm
}
