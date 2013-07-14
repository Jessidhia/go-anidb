// Attempt at high level client library for AniDB's APIs
package anidb

import (
	"time"
)

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
