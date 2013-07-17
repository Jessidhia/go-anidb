package misc_test

import (
	"fmt"
	"github.com/Kovensky/go-anidb/misc"
)

func ExampleEpisodeRange_Merge() {
	a := misc.ParseEpisodeRange("5-7")
	b := misc.ParseEpisodeRange("8-12")
	fmt.Println(a.Merge(b)) // 5-7 + 8-12

	b = misc.ParseEpisodeRange("3-6")
	fmt.Println(a.Merge(b)) // 5-7 + 3-6

	b = misc.ParseEpisodeRange("10-12")
	fmt.Println(a.Merge(b)) // 5-7 + 10-12 (invalid, not touching)

	b = misc.ParseEpisodeRange("S1-S3")
	fmt.Println(a.Merge(b)) // 5-7 + S1-S3 (invalid, different types)

	a = misc.ParseEpisodeRange("S3-S10")
	fmt.Println(a.Merge(b)) // S3-S10 + S1-S3

	// Output:
	// 05-12
	// 3-7
	// <nil>
	// <nil>
	// S01-S10
}

func ExampleEpisodeRange_PartialMerge() {
	a := misc.ParseEpisodeRange("2.1-2.3")
	b := misc.ParseEpisodeRange("3.0")
	fmt.Println(a.Merge(b)) // 2.1-2.3 + 3.0

	b = misc.ParseEpisodeRange("3.1")
	fmt.Println(a.Merge(b)) // 2.1-2.3 + 3.1

	b = misc.ParseEpisodeRange("1")
	fmt.Println(a.Merge(b)) // 2.1-2.3 + 1

	a = misc.ParseEpisodeRange("2.0-2.3")
	fmt.Println(a.Merge(b)) // 2.0-2.3 + 1

	// Output:
	// 2.1-3.0
	// <nil>
	// <nil>
	// 1-2.3
}

func ExampleEpisodeRange_Split() {
	a := misc.ParseEpisodeRange("1.0-1.3")
	b := misc.ParseEpisode("1.2")
	fmt.Println(a.Split(b))

	b = misc.ParseEpisode("1")
	fmt.Println(a.Split(b))

	a = misc.ParseEpisodeRange("1.1-2")
	fmt.Println(a.Split(b))

	b = misc.ParseEpisode("2")
	fmt.Println(a.Split(b))

	a = misc.ParseEpisodeRange("1-10")
	fmt.Println(a.Split(b))

	b = misc.ParseEpisode("4")
	fmt.Println(a.Split(b))

	// Output:
	// [1.0-1.1 1.3]
	// [<nil> <nil>]
	// [<nil> 2]
	// [1.1 <nil>]
	// [1 03-10]
	// [1-3 05-10]
}
