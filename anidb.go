// Attempt at high level client library for AniDB's APIs
package anidb

import (
	"io/ioutil"
	"log"
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
	Logger  *log.Logger   // Logger where HTTP/UDP traffic is logged

	udp *udpWrap
}

// Initialises a new AniDB.
func NewAniDB() *AniDB {
	ret := &AniDB{
		Timeout: 45 * time.Second,
		Logger:  log.New(ioutil.Discard, "", log.LstdFlags),
	}
	ret.udp = newUDPWrap(ret)
	return ret
}

func (adb *AniDB) User() *User {
	if adb != nil && adb.udp != nil {
		if adb.udp.user != nil {
			return adb.udp.user
		} else if adb.udp.credentials != nil {
			// see if we can get it from the cache (we don't care if it's stale)
			adb.udp.user = UserByName(decrypt(adb.udp.credentials.username))
			if adb.udp.user != nil {
				return adb.udp.user
			}
			// we have to go through the slow path
			adb.udp.user = <-adb.GetUserByName(decrypt(adb.udp.credentials.username))
			return adb.udp.user
		}
	}
	return nil
}
