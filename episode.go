package anidb

import (
	"github.com/Kovensky/go-anidb/misc"
	"time"
)

type Episode struct {
	EID EID // The Episode ID.
	AID AID // The Anime ID this Episode belongs to.

	// Type, Number
	misc.Episode

	Length time.Duration // rounded somehow to minutes

	AirDate *time.Time // The original release date, if available.
	Rating  Rating     // Episode-specific ratings.

	Titles UniqueTitleMap // Map with a title for each language

	Cached time.Time // When the data was retrieved from the server
}

type Episodes []*Episode
