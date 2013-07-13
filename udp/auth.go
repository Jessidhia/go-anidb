package udpapi

import (
	"strings"
)

// Authenticates the supplied user with the supplied password. Blocks until we have a reply.
// Needed before almost any other API command can be used.
//
// If the udpKey is not "", then the connection will be encrypted, but the protocol's
// encryption uses the VERY weak ECB mode.
//
// http://wiki.anidb.net/w/UDP_API_Definition#AUTH:_Authing_to_the_AnimeDB
//
// http://wiki.anidb.net/w/UDP_API_Definition#ENCRYPT:_Start_Encrypted_Session
func (a *AniDBUDP) Auth(user, password, udpKey string) (err error) {
	if a.session != "" {
		if err = (<-a.Uptime()).Error(); err == nil {
			return nil
		}
	}

	a.session = ""
	if udpKey != "" {
		if err = a.encrypt(user, udpKey); err != nil {
			return err
		}
	}
	r := <-a.SendRecv("AUTH", ParamMap{
		"user":      user,
		"pass":      password,
		"protover":  3,
		"client":    "goanidbudp",
		"clientver": 1,
		"nat":       1,
		"comp":      1,
		"enc":       "UTF-8",
	})
	switch r.Code() {
	case 200, 201:
		f := strings.Fields(r.Text())
		a.session = f[0]
	}
	return r.Error()
}

// Ends the API session. Blocks until we have confirmation.
//
// http://wiki.anidb.net/w/UDP_API_Definition#LOGOUT:_Logout
func (a *AniDBUDP) Logout() (err error) {
	r := <-a.SendRecv("LOGOUT", ParamMap{})
	a.session = ""
	return r.Error()
}

func (a *AniDBUDP) encrypt(user, udpKey string) (err error) {
	if reply := <-a.SendRecv("ENCRYPT", ParamMap{"user": user, "type": 1}); reply.Error() != nil {
		return reply.Error()
	} else {
		switch reply.Code() {
		case 209:
			salt := []byte(strings.Fields(reply.Text())[0])

			// Yes, AniDB works in ECB mode
			a.ecb = newECBState(udpKey, salt)
		}
		return reply.Error()
	}
}
