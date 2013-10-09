// Low-level wrapper around the AniDB HTTP API.
// Only implements the 'anime' and 'categorylist' requests.
//
// This wrapper does not implement caching. The API requires
// aggressive caching.
//
// http://wiki.anidb.info/w/HTTP_API_Definition
package httpapi

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const (
	AniDBImageBaseURL = "http://img7.anidb.net/pics/anime/" // Base URL for the various Pictures in the response

	DateFormat = "2006-01-02" // Use to convert the various YYYY-MM-DD timestamps to a time.Time.

	// Base URLs for the various resources.
	// Meant for use with fmt.Sprintf.
	ANNFormat          = "http://www.animenewsnetwork.com/encyclopedia/anime.php?id=%v" // Type 1
	MyAnimeListFormat  = "http://myanimelist.net/anime/%v"                              // Type 2
	AnimeNfoFormat     = "http://www.animenfo.com/animetitle,%v,%v,a.html"              // Type 3
	_                                                                                   // Type 4
	_                                                                                   // Type 5
	WikiEnglishFormat  = "http://en.wikipedia.org/wiki/%v"                              // Type 6
	WikiJapaneseFormat = "http://ja.wikipedia.org/wiki/%v"                              // Type 7
	SyoboiFormat       = "http://cal.syoboi.jp/tid/%v/time"                             // Type 8
	AllCinemaFormat    = "http://www.allcinema.net/prog/show_c.php?num_c=%v"            // Type 9
	AnisonFormat       = "http://anison.info/data/program/%v.html"                      // Type 10
	LainGrJpFormat     = "http://lain.gr.jp/%v"                                         // Type 11
	_                                                                                   // Type 12
	_                                                                                   // Type 13
	VNDBFormat         = "http://vndb.org/v%v"                                          // Type 14
	MaruMeganeFormat   = "http://www.anime.marumegane.com/%v.html"                      // Type 15
	_                                                                                   // Type 16
	TVAnimationMuseum  = "http://home-aki.cool.ne.jp/anime-list/%s.htm"                 // Type 17 (broken)
	_                                                                                   // Type 18
	WikiKoreanformat   = "http://ko.wikipedia.org/wiki/%v"                              // Type 19
	WikiChineseFormat  = "http://zh.wikipedia.org/wiki/%v"                              // Type 20
)

const (
	aniDBHTTPAPIBaseURL = "http://api.anidb.net:9001/httpapi"
	aniDBProtoVer       = 1
	clientStr           = "goanidbhttp"
	clientVer           = 1
)

// Requests information about the given Anime ID.
func GetAnime(AID int) (a Anime, err error) {
	if res, err := doRequest("anime", reqMap{"aid": AID}); err != nil {
		return a, err
	} else {
		dec := xml.NewDecoder(res.Body)
		err = dec.Decode(&a)
		res.Body.Close()

		a.Error = strings.TrimSpace(a.Error)

		title := ""
		for _, t := range a.Titles {
			if t.Type == "main" {
				title = t.Title
				break
			}
		}

		for _, r := range a.Resources {
			switch r.Type {
			case 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 14, 15, 17, 19, 20:
				// documentation knows about these
			default:
				log.Printf("HTTP -- Anime %d (%s) has unknown resource type %d", a.ID, title, r.Type)
				log.Printf("HTTP -- Type %d external entities: %#v", r.Type, r.ExternalEntity)
			}
		}

		return a, err
	}
}

type reqMap map[string]interface{}

func doRequest(request string, reqMap reqMap) (*http.Response, error) {
	v := url.Values{}
	v.Set("protover", fmt.Sprint(aniDBProtoVer))
	v.Set("client", clientStr)
	v.Set("clientver", fmt.Sprint(clientVer))
	v.Set("request", request)

	for k, val := range reqMap {
		v.Add(k, fmt.Sprint(val))
	}

	u, _ := url.Parse(aniDBHTTPAPIBaseURL)
	u.RawQuery = v.Encode()
	return http.Get(u.String())
}

// Title with language and type identifier.
//
// Title with Lang = ja, Type = official is the official Kanji title.
//
// Title with Lang = x-jat, Type = main is the romanized version, also known in other APIs as the Primary Title.
type AnimeTitle struct {
	Lang  string `xml:"lang,attr"` // Language in ISO-ish format
	Type  string `xml:"type,attr"` // "official", "short", etc
	Title string `xml:",chardata"`
}

type RelatedAnime struct {
	ID    int    `xml:"id,attr"`   // AID of the related anime
	Type  string `xml:"type,attr"` // "prequel", "sequel", etc
	Title string `xml:",chardata"` // Primary title of the related anime
}

type SimilarAnime struct {
	ID       int    `xml:"id,attr"`       // AID of the similar anime
	Approval int    `xml:"approval,attr"` // How many users have approved of this connection
	Total    int    `xml:"total,attr"`    // Total of votes in this connection
	Title    string `xml:",chardata"`     // Primary title of the recommended anime
}

type Recommendation struct {
	Type string `xml:"type,attr"` // "Recommended", "Must See", etc
	ID   int    `xml:"uid,attr"`  // User ID of the recommending user
	Text string `xml:",chardata"` // Text of the user's recommendation
}

type Creator struct {
	ID   int    `xml:"id,attr"` // Creator ID
	Type string `xml:"type,attr"`
	Name string `xml:",chardata"` // Always romaji
}

// Separate from regular Rating because the XML structure is different.
type AnimeRating struct {
	Count  int     `xml:"count,attr"` // Amount of votes/reviews
	Rating float32 `xml:",chardata"`  // Average
}

type AnimeRatings struct {
	Permanent AnimeRating `xml:"permanent"` // Votes from people who watched everything
	Temporary AnimeRating `xml:"temporary"` // Votes from people who are still watching it
	Review    AnimeRating `xml:"review"`    // Votes from reviews
}

type Category struct {
	ID       int  `xml:"id,attr"`       // Category ID
	ParentID int  `xml:"parentid,attr"` // ID of the parent category
	R18      bool `xml:"hentai,attr"`   // Whether the category represents porn works or not
	Weight   int  `xml:"weight,attr"`   // Weight of the category for this anime

	Name        string `xml:"name"`        // Category name
	Description string `xml:"description"` // Category description
}

type ExternalEntity struct {
	Identifiers []string `xml:"identifier"`
	URL         []string `xml:"url"` // Used for some types
}

// Completely undocumented.
// Most entries just have one or two numbers as Identifiers.
//
// Empiric documentation:
//
// Type 1 is the ANN id.
//
// Type 2 is the MyAnimeList ID.
//
// Type 3 is the AnimeNfo ID tuple.
//
// Type 4 is the official japanese webpage. URL may contain additional URLs (official PV, etc)
//
// Type 5 is the official english webpage.
//
// Type 6 is the english wikipedia page name.
//
// Type 7 is the japanese wikipedia page name.
//
// Type 8 is the cal.syoboi.jp schedule ID.
//
// Type 9 is the AllCinema ID.
//
// Type 10 is the anison.info ID.
//
// Type 11 is the lain.gr.jp path.
//
// Type 14 is the VNDB ID.
//
// Type 15 is the MaruMegane ID.
//
// Type 17 would be the TV Animation Museum identifier, but the website is no more.
//
// Type 19 is the korean wikipedia page name.
//
// Type 20 is the chinese wikipedia page name.
type Resource struct {
	Type           int              `xml:"type,attr"`
	ExternalEntity []ExternalEntity `xml:"externalentity"`
}

type Tag struct {
	ID            int    `xml:"id,attr"`            // Tag ID
	Approval      int    `xml:"approval,attr"`      // How many users have approved of the tag
	Spoiler       bool   `xml:"localspoiler,attr"`  // undocumented
	GlobalSpoiler bool   `xml:"globalspoiler,attr"` // undocumented
	Updated       string `xml:"update,attr"`        // YYYY-MM-DD

	Name  string `xml:"name"`  // Tag name
	Count int    `xml:"count"` // undocumented
}

type Seiyuu struct {
	ID      int    `xml:"id,attr"`      // Creator ID
	Name    string `xml:",chardata"`    // Always romaji
	Picture string `xml:"picture,attr"` // Picture basename; combine with AniDBImageBaseURL for full URL
}

type Character struct {
	ID      int    `xml:"id,attr"`     // Character ID
	Type    string `xml:"type,attr"`   // "main character in", "secondary cast in", "appears in"
	Updated string `xml:"update,attr"` // YYYY-MM-DD

	Rating        Rating `xml:"rating"`
	Name          string `xml:"name"`   // Always romaji
	Gender        string `xml:"gender"` // "male", "female", "unknown", sometimes blank
	Description   string `xml:"description"`
	CharacterType string `xml:"charactertype"` // "Character", "Organization", "Vessel", etc
	Episodes      string `xml:"episodes"`      // List of episodes where character appears
	Picture       string `xml:"picture"`       // Picture basename; combine with AniDBImageBaseURL for full URL

	Seiyuu *Seiyuu `xml:"seiyuu"` // The voice actor, if present
}

type Characters []Character // Implements sort.Interface; groups by Type and sorts by Name

type EpisodeTitle struct {
	Lang  string `xml:"lang,attr"`
	Title string `xml:",chardata"`
}

type Rating struct {
	Votes  int     `xml:"votes,attr"`
	Rating float32 `xml:",chardata"`
}

type EpNo struct {
	Type int    `xml:"type,attr"` // 1 for regular episodes, 2 for specials, etc
	EpNo string `xml:",chardata"` // Not necessarily a plain integer; may be prefixed by a single letter indicating the Type
}

type Episode struct {
	ID      int    `xml:"id,attr"`     // Episode ID
	Updated string `xml:"update,attr"` // YYYY-MM-DD

	EpNo    EpNo           `xml:"epno"`
	Length  int            `xml:"length"`  // Length in minutes (rounding method undocumented)
	AirDate string         `xml:"airdate"` // YYYY-MM-DD
	Rating  Rating         `xml:"rating"`
	Titles  []EpisodeTitle `xml:"title"`
}

type Episodes []Episode // Implements sort.Interface; groups by EpNo.Type, orders by the integer portion of EpNo.EpNo

type Anime struct {
	Error string `xml:",chardata"` // API request encountered an error if this is not ""

	ID  int  `xml:"id,attr"`         // AID of the anime
	R18 bool `xml:"restricted,attr"` // Whether the anime is considered porn

	Type         string `xml:"type"`         // "TV Series", "Movie", "OVA", etc
	EpisodeCount int    `xml:"episodecount"` // Unreliable, has a set value even when the total number is unknown
	StartDate    string `xml:"startdate"`    // YYYY-MM-DD
	EndDate      string `xml:"enddate"`      // YYYY-MM-DD

	Titles       []AnimeTitle   `xml:"titles>title"`
	RelatedAnime []RelatedAnime `xml:"relatedanime>anime"`
	SimilarAnime []SimilarAnime `xml:"similaranime>anime"`

	Recommendations []Recommendation `xml:"recommendations>recommendation"`

	URL string `xml:"url"` // Official URL

	Creators []Creator `xml:"creators>name"`

	Description string `xml:"description"`

	Ratings AnimeRatings `xml:"ratings"`

	Picture string `xml:"picture"` // Picture basename; combine with AniDBImageBaseURL for full URL

	Categories []Category `xml:"categories>category"`  // Unsorted
	Resources  []Resource `xml:"resources>resource"`   // undocumented
	Tags       []Tag      `xml:"tags>tag"`             // Unsorted
	Characters Characters `xml:"characters>character"` // Unsorted
	Episodes   Episodes   `xml:"episodes>episode"`     // Unsorted
}
