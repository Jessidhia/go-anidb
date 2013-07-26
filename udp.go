package anidb

import (
	"github.com/Kovensky/go-anidb/udp"
	"sync"
	"time"
)

const banDuration = 30*time.Minute + 1*time.Second

// Returns whether the last UDP API access returned a 555 BANNED message.
func Banned() bool {
	stat, err := Cache.Stat("banned")
	if err != nil {
		return false
	}

	switch ts := stat.ModTime(); {
	case ts.IsZero():
		return false
	case time.Now().Sub(ts) > banDuration:
		return false
	default:
		return true
	}
}

func setBanned() {
	Cache.Touch("banned")
}

type paramSet struct {
	cmd    string
	params paramMap
	ch     chan udpapi.APIReply
}

type udpWrap struct {
	*udpapi.AniDBUDP

	adb *AniDB

	sendLock    sync.Mutex
	sendQueueCh chan paramSet

	credLock    sync.Mutex
	credentials *credentials
	connected   bool

	user *User
}

func newUDPWrap(adb *AniDB) *udpWrap {
	u := &udpWrap{
		AniDBUDP:    udpapi.NewAniDBUDP(),
		adb:         adb,
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

func (udp *udpWrap) logRequest(set paramSet) {
	switch set.cmd {
	case "AUTH":
		udp.adb.Logger.Printf("UDP>>> AUTH user=%s\n", set.params["user"])
	default:
		udp.adb.Logger.Printf("UDP>>> %s %s\n", set.cmd, udpapi.ParamMap(set.params).String())
	}
}

func (udp *udpWrap) logReply(reply udpapi.APIReply) {
	udp.adb.Logger.Printf("UDP<<< %d %s\n", reply.Code(), reply.Text())
}

func (udp *udpWrap) sendQueue() {
	initialWait := 5 * time.Second
	wait := initialWait
	for set := range udp.sendQueueCh {
	Retry:
		if Banned() {
			set.ch <- bannedReply
			close(set.ch)
			continue
		}

		udp.logRequest(set)
		reply := <-udp.AniDBUDP.SendRecv(set.cmd, udpapi.ParamMap(set.params))

		if reply.Error() == udpapi.TimeoutError {
			// retry
			wait = wait * 2
			if wait > time.Minute {
				wait = time.Minute
			}
			udp.adb.Logger.Printf("UDP--- Timeout; waiting %s before retry", wait)

			delete(set.params, "s")
			delete(set.params, "tag")

			time.Sleep(wait)
			goto Retry
		}
		udp.logReply(reply)

		wait = initialWait

		switch reply.Code() {
		case 403, 501, 506: // not logged in, or session expired
			if r := udp.ReAuth(); r.Error() == nil {
				// retry

				delete(set.params, "s")
				delete(set.params, "tag")

				goto Retry
			}
		case 503, 504: // client library rejected
			panic(reply.Error())
		// 555: IP (and user, possibly client) temporarily banned
		// 601: Server down (treat the same as a ban)
		case 555, 601:
			setBanned()
		}
		set.ch <- reply
		close(set.ch)
	}
}

func (udp *udpWrap) SendRecv(cmd string, params paramMap) <-chan udpapi.APIReply {
	ch := make(chan udpapi.APIReply, 1)

	udp.sendLock.Lock()
	defer udp.sendLock.Unlock()

	if Banned() {
		ch <- bannedReply
		close(ch)
		return ch
	}

	if !udp.connected {
		if r := udp.ReAuth(); r.Error() != nil {
			ch <- r
			close(ch)
			return ch
		}
	}

	udp.sendQueueCh <- paramSet{
		cmd:    cmd,
		params: params,
		ch:     ch,
	}

	return ch
}
