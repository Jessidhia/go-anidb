package udpapi

import (
	"strconv"
	"time"
)

type UptimeReply struct {
	APIReply
	Uptime time.Duration
}

// Retrieves the server's uptime. The recommended way to verify if a session
// is valid.
//
// Returns a channel through which the eventual response will be sent.
//
// http://wiki.anidb.net/w/UDP_API_Definition#UPTIME:_Retrieve_Server_Uptime
func (a *AniDBUDP) Uptime() <-chan UptimeReply {
	ch := make(chan UptimeReply, 2)
	go func() {
		reply := <-a.SendRecv("UPTIME", ParamMap{})

		r := UptimeReply{APIReply: reply}
		if r.Error() == nil {
			uptime, _ := strconv.ParseInt(reply.Lines()[1], 10, 32)
			r.Uptime = time.Duration(uptime) * time.Millisecond
		}
		ch <- r
		close(ch)
	}()
	return ch
}

type PingReply struct {
	APIReply
	Port uint16 // This client's local UDP port
}

// Simple echo command. The recommended way to verify if the server
// is alive and responding. Does not require authentication.
//
// Returns a channel through which the eventual response will be sent.
//
// http://wiki.anidb.net/w/UDP_API_Definition#PING:_Ping_Command
func (a *AniDBUDP) Ping() <-chan PingReply {
	ch := make(chan PingReply, 2)
	go func() {
		reply := <-a.SendRecv("PING", ParamMap{"nat": 1})

		r := PingReply{APIReply: reply}
		if r.Error() == nil {
			port, _ := strconv.ParseUint(reply.Lines()[1], 10, 16)
			r.Port = uint16(port)
		}
		ch <- r
		close(ch)
	}()
	return ch
}
