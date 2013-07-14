package anidb

import (
	"github.com/Kovensky/go-anidb/udp"
	"sync"
	"time"
)

var banTime time.Time
var banTimeLock sync.Mutex

const banDuration = 30*time.Minute + 1*time.Second

// Returns whether the last UDP API access returned a 555 BANNED message.
func Banned() bool {
	banTimeLock.Lock()
	banTimeLock.Unlock()

	return _banned()
}

func _banned() bool {
	return time.Now().Sub(banTime) > banDuration
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
	return &udpapi.APIError{Code: 555, Desc: "BANNED"}
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
			banTimeLock.Lock()

			banTime = time.Now()

			banTimeLock.Unlock()
		}
		set.ch <- reply
		close(set.ch)
	}
}

func (udp *udpWrap) SendRecv(cmd string, params paramMap) <-chan udpapi.APIReply {
	ch := make(chan udpapi.APIReply, 1)

	banTimeLock.Lock()
	defer banTimeLock.Unlock()
	if _banned() {
		banTime = time.Time{}
	} else {
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
