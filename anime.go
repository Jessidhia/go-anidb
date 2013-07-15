package anidb

import (
	"github.com/Kovensky/go-anidb/misc"
	"strconv"
	"time"
)

// See the constants list for the valid values.
type AnimeType string

const (
	AnimeTypeTVSeries   = AnimeType("TV Series")  // Anime was a regular TV broadcast series
	AnimeTypeTVSpecial  = AnimeType("TV Special") // Anime was broadcast on TV as a special
	AnimeTypeMovie      = AnimeType("Movie")      // Anime was a feature film
	AnimeTypeOVA        = AnimeType("OVA")        // Anime was released direct-to-video
	AnimeTypeWeb        = AnimeType("Web")        // Anime was released through online streaming or downloads
	AnimeTypeMusicVideo = AnimeType("Music Video")
)

type Rating struct {
	Rating    float32
	VoteCount int
}

type Resource []string

// Links to third party websites
type Resources struct {
	AniDB,
	ANN,
	MyAnimeList,
	AnimeNfo,
	OfficialJapanese,
	OfficialEnglish,
	WikipediaEnglish,
	WikipediaJapanese,
	SyoboiSchedule,
	AllCinema,
	Anison,
	VNDB,
	MaruMegane Resource
}

type UniqueTitleMap map[Language]string
type TitleMap map[Language][]string

type Anime struct {
	AID AID  // The Anime ID.
	R18 bool // Whether this anime is considered porn.

	Type          AnimeType    // Production/distribution type.
	TotalEpisodes int          // Total number of regular episodes.
	EpisodeCount  EpisodeCount // Known numbers of the various types of episodes.

	StartDate time.Time // Date of first episode release, if available.
	EndDate   time.Time // Date of last episode release, if available.

	PrimaryTitle   string         // The primary title in the database; almost always a romanization of the Japanese title.
	OfficialTitles UniqueTitleMap // The official title for each language.
	ShortTitles    TitleMap       // Shortcut titles used for searches
	Synonyms       TitleMap       // Synonyms for each language, or unofficial titles

	OfficialURL string // URL for original official website.
	Picture     string // URL for the page picture on AniDB.

	Description string

	Votes          Rating // Votes from people who watched the whole thing.
	TemporaryVotes Rating // Votes from people who are still watching this.
	Reviews        Rating // Votes from reviewers.

	Episodes Episodes // List of episodes.

	Awards    []string
	Resources Resources

	Incomplete bool      // Set if the UDP API part of the query failed.
	Updated    time.Time // When the data was last modified in the server.
	Cached     time.Time // When the data was retrieved from the server.
}

type EpisodeCount struct {
	RegularCount int
	SpecialCount int
	CreditsCount int
	OtherCount   int
	TrailerCount int
	ParodyCount  int
}

// Convenience method that runs AnimeByID on the result of
// SearchAnime.
func (adb *AniDB) AnimeByName(name string) <-chan *Anime {
	return adb.AnimeByID(SearchAnime(name))
}

// Convenience method that runs AnimeByID on the result of
// SearchAnimeFold.
func (adb *AniDB) AnimeByNameFold(name string) <-chan *Anime {
	return adb.AnimeByID(SearchAnimeFold(name))
}

// Returns a list of all Episodes in this Anime's Episodes list
// that are contained by the given EpisodeContainer.
func (a *Anime) EpisodeList(c misc.EpisodeContainer) (eps []*Episode) {
	if a == nil || c == nil {
		return nil
	}

	for i, e := range a.Episodes {
		if c.ContainsEpisodes(&e.Episode) {
			eps = append(eps, a.Episodes[i])
		}
	}
	return
}

// Searches for the given Episode in this Anime's Episodes list
// and returns the match.
//
// Returns nil if there is no match.
func (a *Anime) Episode(ep *misc.Episode) *Episode {
	switch list := a.EpisodeList(ep); len(list) {
	case 0:
		return nil
	case 1:
		return list[0]
	default:
		panic("Single episode search returned more than one result")
	}
}

// Convenience method that parses the string into an Episode
// before doing the Episode search.
func (a *Anime) EpisodeByString(name string) *Episode {
	return a.Episode(misc.ParseEpisode(name))
}

// Convenience method that parses the int into an Episode
// before doing the Episode search.
//
// Only works with regular (i.e. not special, etc) episodes.
func (a *Anime) EpisodeByNumber(number int) *Episode {
	return a.EpisodeByString(strconv.Itoa(number))
}
