package misc_test

import (
	"fmt"
	"github.com/Kovensky/go-anidb/misc"
)

func ExampleEpisodeList_Simplify() {
	a := misc.ParseEpisodeList("1,2,3,5,10-14,13-15,,S3-S6,C7-C10,S1,S7,S8-")
	fmt.Println(a.Simplify())

	// Output: 01-03,05,10-15,S1,S3-,C07-C10
}

func ExampleEpisodeList_Add() {
	a := misc.ParseEpisodeList("1-3")
	a.Add(misc.ParseEpisode("3.1"))
	fmt.Println(a)

	a.Add(misc.ParseEpisode("4.0"))
	fmt.Println(a)

	a.Add(misc.ParseEpisode("4"))
	fmt.Println(a)

	a.Add(misc.ParseEpisode("5.1"))
	fmt.Println(a)

	a.Add(misc.ParseEpisode("6"))
	fmt.Println(a)

	// Output:
	// 1-3
	// 1-4.0
	// 1-4
	// 1-4,5.1
	// 1-4,5.1,6
}
