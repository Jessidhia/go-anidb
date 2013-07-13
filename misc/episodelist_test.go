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
