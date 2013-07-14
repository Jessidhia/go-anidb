package anidb

import (
	"encoding/gob"
	"github.com/Kovensky/go-anidb/udp"
	"time"
)

func init() {
	gob.RegisterName("*github.com/Kovensky/go-anidb.banCache", &banCache{})
}

const banDuration = 30*time.Minute + 1*time.Second

type banCache struct{ time.Time }

func (c *banCache) Touch() {
	c.Time = time.Now()
}
func (c *banCache) IsStale() bool {
	return time.Now().Sub(c.Time) > banDuration
}

// Returns whether the last UDP API access returned a 555 BANNED message.
func Banned() bool {
	var banTime banCache
	cache.Get(&banTime, "banned")

	stale := banTime.IsStale()
	if stale {
		cache.Delete("banned")
	}
	return !stale
}

func setBanned() {
	cache.Set(&banCache{}, "banned")
}

type paramSet struct {
	cmd    string
	params paramMap
	ch     chan udpapi.APIReply
}

type udpWrap struct {
	*udpapi.AniDBUDP

	sendQueueCh chan paramSet

	credentials *credentials
	connected   bool
}

func newUDPWrap() *udpWrap {
	u := &udpWrap{
		AniDBUDP:    udpapi.NewAniDBUDP(),
		sendQueueCh: make(chan paramSet, 10),
	}
	go u.sendQueue()
	return u
}

type paramMap udpapi.ParamMap // shortcut

type noauthAPIReply struct {
	udpapi.APIReply
}

func (r *noauthAPIReply) Code() int {
	return 501
}

func (r *noauthAPIReply) Text() string {
	return "LOGIN FIRST"
}

func (r *noauthAPIReply) Error() error {
	return &udpapi.APIError{Code: r.Code(), Desc: r.Text()}
}

type bannedAPIReply struct {
	udpapi.APIReply
}

func (r *bannedAPIReply) Code() int {
	return 555
}
func (r *bannedAPIReply) Text() string {
	return "BANNED"
}
func (r *bannedAPIReply) Error() error {
	return &udpapi.APIError{Code: r.Code(), Desc: r.Text()}
}

var bannedReply udpapi.APIReply = &bannedAPIReply{}

func (udp *udpWrap) sendQueue() {
	for set := range udp.sendQueueCh {
		reply := <-udp.AniDBUDP.SendRecv(set.cmd, udpapi.ParamMap(set.params))

		if reply.Error() == udpapi.TimeoutError {
			// retry
			go func(set paramSet) { udp.sendQueueCh <- set }(set)
			continue
		}

		switch reply.Code() {
		case 403, 501, 506: // not logged in, or session expired
			if err := udp.ReAuth(); err == nil {
				// retry
				go func(set paramSet) { udp.sendQueueCh <- set }(set)
				continue
			}
		case 503, 504: // client library rejected
			panic(reply.Error())
		case 555: // IP (and user, possibly client) temporarily banned
			setBanned()
		}
		set.ch <- reply
		close(set.ch)
	}
}

func (udp *udpWrap) SendRecv(cmd string, params paramMap) <-chan udpapi.APIReply {
	ch := make(chan udpapi.APIReply, 1)
	if udp.credentials == nil {
		ch <- &noauthAPIReply{}
		close(ch)
		return ch
	}

	if Banned() {
		ch <- bannedReply
		close(ch)
		return ch
	}

	udp.sendQueueCh <- paramSet{
		cmd:    cmd,
		params: params,
		ch:     ch,
	}

	return ch
}
