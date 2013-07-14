package anidb

import (
	"time"
)

type Language string

var (
	// Default durations for the various caches.
	// Used by the IsStale methods.
	DefaultCacheDuration = 7 * 24 * time.Hour
	AnimeCacheDuration   = DefaultCacheDuration
	EpisodeCacheDuration = DefaultCacheDuration
	GroupCacheDuration   = 4 * DefaultCacheDuration // They don't change that often.
	FileCacheDuration    = 8 * DefaultCacheDuration // These change even less often.

	InvalidKeyCacheDuration = 1 * time.Hour

	// Used when the UDP API Anime query fails, but the HTTP API query succeeds.
	AnimeIncompleteCacheDuration = 24 * time.Hour

	// Used when there's some data missing on a file.
	// Usually happens because the AVDump data hasn't been merged with the database
	// yet, which is done on a daily cron job.
	FileIncompleteCacheDuration = 24 * time.Hour
)
