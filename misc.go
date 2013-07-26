package anidb

import (
	"github.com/Kovensky/go-fscache"
	"strconv"
	"strings"
	"time"
)

type MyListStats struct {
	Anime         int
	Episodes      int
	Files         int
	Filesize      int64
	AddedAnime    int
	AddedEpisodes int
	AddedFiles    int
	AddedGroups   int

	Leech             float32 // rate of Files to AddedFiles
	Glory             float32 // meaning undocumented
	ViewedPctDatabase float32
	MyListPctDatabase float32
	AnimePctDatabase  float32 // Only valid if the titles database is loaded
	ViewedPctMyList   float32
	ViewedEpisodes    int
	Votes             int
	Reviews           int

	ViewedTime time.Duration

	Cached time.Time
}
