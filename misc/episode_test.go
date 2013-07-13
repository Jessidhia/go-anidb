package misc_test

import (
	"fmt"
	"github.com/Kovensky/go-anidb/misc"
)

func ExampleParseEpisode() {
	fmt.Printf("%#v\n", misc.ParseEpisode("1"))
	fmt.Printf("%#v\n", misc.ParseEpisode("S2"))
	fmt.Printf("%#v\n", misc.ParseEpisode("03"))
	fmt.Printf("%#v\n", misc.ParseEpisode("")) // invalid episode

	// Output:
	// &misc.Episode{Type:1, Number:1}
	// &misc.Episode{Type:2, Number:2}
	// &misc.Episode{Type:1, Number:3}
	// (*misc.Episode)(nil)
}

//	ParseEpisodeRange("1")     <=> ep := ParseEpisode("1");
//		&EpisodeRange{Type: EpisodeTypeRegular, Start: ep, End: ep}
//	ParseEpisodeRange("S1-")   <=>
//		&EpisodeRange{Type: EpisodeTypeSpecial, Start: ParseEpisode("S1")}
//	ParseEpisodeRange("T1-T3") <=>
//		&EpisodeRange{Type: EpisodeTypeTrailer, Start: ParseEpisode("T1"), End: ParseEpisode("T3")}
//	ParseEpisodeRange("5-S3")  <=> nil // different episode types in range
//	ParseEpisodeRange("")      <=> nil // invalid start of range

func ExampleParseEpisodeRange() {
	fmt.Println(misc.ParseEpisodeRange("01"))
	fmt.Println(misc.ParseEpisodeRange("S1-")) // endless range
	fmt.Println(misc.ParseEpisodeRange("T1-T3"))
	fmt.Println(misc.ParseEpisodeRange("5-S3")) // different episode types in range
	fmt.Println(misc.ParseEpisodeRange(""))     // invalid start of range

	// Output:
	// 1
	// S1-
	// T1-T3
	// <nil>
	// <nil>
}
