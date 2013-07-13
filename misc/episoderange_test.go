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
