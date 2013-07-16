// Attempt at high level client library for AniDB's APIs
package anidb

import (
	"time"
)

// Main struct for the client, contains all non-shared state.
//
// All ObjectByKey methods (AnimeByID, GroupByName, etc) first try to read
// from the cache. If the sought object isn't cached, or if the cache is
// stale, then the appropriate API is queried.
//
// Queries return their results using channels. Most queries only have one result,
// but some have 0 or more. All result channels are closed after sending their data.
//
// Queries that depend on the UDP API can't be used without first authenticating
// to the API server. This uses the credentials stored by SetCredentials, or
// by a previous Auth() call.
type AniDB struct {
	Timeout time.Duration // Timeout for the various calls (default: 45s)

	udp *udpWrap
}

// Initialises a new AniDB.
func NewAniDB() *AniDB {
	return &AniDB{
		Timeout: 45 * time.Second,
		udp:     newUDPWrap(),
	}
}
