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
	// &misc.Episode{Type:1, Number:1, Part:-1, Parts:0}
	// &misc.Episode{Type:2, Number:2, Part:-1, Parts:0}
	// &misc.Episode{Type:1, Number:3, Part:-1, Parts:0}
	// (*misc.Episode)(nil)
}

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

func ExamplePartialEpisode() {
	eps := []*misc.Episode{
		misc.ParseEpisode("1.0"),
		misc.ParseEpisode("1.1"),
	}
	for _, ep := range eps {
		fmt.Printf("%#v %s\n", ep, ep)
	}
	for _, ep := range eps {
		ep.Parts = 2
		fmt.Printf("%s\n", ep)
	}

	// Output:
	// &misc.Episode{Type:1, Number:1, Part:0, Parts:0} 1.0
	// &misc.Episode{Type:1, Number:1, Part:1, Parts:0} 1.1
	// 1.00
	// 1.50
}
