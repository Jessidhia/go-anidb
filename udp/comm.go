// Low-level AniDB UDP API client library
//
// Implements the commands essential to setting up and tearing down an API connection,
// as well as an asynchronous layer. Throttles sends internally according to API spec.
//
// This library doesn't implement caching; beware since aggressive caching is an
// implementation requirement. Not doing so can get you banned.
package udpapi

import (
	"bytes"
	"compress/zlib"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	AniDBUDPServer = "api.anidb.net"
	AniDBUDPPort   = 9000
)

type AniDBUDP struct {
	// Interval between keep-alive packets; only sent when PUSH notifications are enabled (default: 20 minutes)
	KeepAliveInterval time.Duration

	// The time to wait before a packet is considered lost (default: 45 seconds)
	Timeout time.Duration

	// Channel where PUSH notifications are sent to
	Notifications chan APIReply

	session string

	conn *net.UDPConn
	ecb  *ecbState

	counter uint16
	ctrLock sync.Mutex

	tagRouter  map[string]chan APIReply
	routerLock sync.RWMutex

	sendCh chan packet

	breakRecv chan bool
	breakSend chan bool

	// notifyState *notifyState
	pingTimer *time.Timer
}

// Creates and initializes the AniDBUDP struct
func NewAniDBUDP() *AniDBUDP {
	c := &AniDBUDP{
		KeepAliveInterval: 20 * time.Minute,
		Timeout:           45 * time.Second,
		Notifications:     make(chan APIReply, 5),
		tagRouter:         make(map[string]chan APIReply),
	}
	return c
}

// Key-value list of parameters.
type ParamMap map[string]interface{}

// Returns a query-like string representation of the ParamMap
func (m ParamMap) String() string {
	keys := make([]string, 0, len(m))
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(m))
	for _, k := range keys {
		parts = append(parts, strings.Join([]string{k, fmt.Sprint((m)[k])}, "="))
	}
	return strings.Join(parts, "&")
}

// Sends the requested query to the AniDB UDP API server.
//
// Returns a channel through which the eventual reply is sent.
//
// See http://wiki.anidb.net/w/UDP_API_Definition for the defined commands.
func (a *AniDBUDP) SendRecv(command string, args ParamMap) <-chan APIReply {
	a.ctrLock.Lock()
	tag := fmt.Sprintf("T%d", a.counter)
	a.counter++
	a.ctrLock.Unlock()

	args["tag"] = tag
	if a.session != "" {
		args["s"] = a.session
	}
	for k, v := range args {
		v = strings.Replace(v, "\n", "<br/>", -1)
		args[k] = strings.Replace(v, "&", "&amp;", -1)
	}

	if err := a.dial(); err != nil {
		ch <- newErrorWrapper(err)
		close(ch)
		return ch
	}

	ch := make(chan APIReply, 1)

	a.routerLock.Lock()
	a.tagRouter[tag] = ch
	a.routerLock.Unlock()

	reply := make(chan APIReply, 1)
	go func() {
		<-a.send(command, args)
		timeout := time.After(a.Timeout)

		select {
		case <-timeout:
			a.routerLock.Lock()
			delete(a.tagRouter, tag)
			a.routerLock.Unlock()
			close(ch)

			reply <- TimeoutError
			close(reply)

			log.Println("!!! Timeout")
		case r := <-ch:
			a.routerLock.Lock()
			delete(a.tagRouter, tag)
			a.routerLock.Unlock()
			close(ch)

			reply <- r
			close(reply)
		}
	}()
	return reply
}

var laddr, _ = net.ResolveUDPAddr("udp4", "0.0.0.0:0")

func (a *AniDBUDP) dial() (err error) {
	if a.conn != nil {
		return nil
	}

	srv := fmt.Sprintf("%s:%d", AniDBUDPServer, AniDBUDPPort)
	if raddr, err := net.ResolveUDPAddr("udp4", srv); err != nil {
		return err
	} else {
		a.conn, err = net.DialUDP("udp4", laddr, raddr)

		if a.breakSend != nil {
			a.breakSend <- true
			<-a.breakSend
		} else {
			a.breakSend = make(chan bool)
		}
		a.sendCh = make(chan packet, 10)
		go a.sendLoop()

		if a.breakRecv != nil {
			a.breakRecv <- true
			<-a.breakRecv
		} else {
			a.breakRecv = make(chan bool)
		}
		go a.recvLoop()
	}
	return err
}

func (a *AniDBUDP) send(command string, args ParamMap) chan bool {
	str := command
	arg := args.String()
	if len(arg) > 0 {
		str = strings.Join([]string{command, arg}, " ")
	}
	log.Println(">>>", str)

	p := makePacket([]byte(str), a.ecb)

	sendPacket(p, a.sendCh)
}

type packet struct {
	b    []byte
	err  error
	sent chan bool
}

func (a *AniDBUDP) sendLoop() {
	for {
		select {
		case <-a.breakSend:
			a.breakSend <- true
			return
		case pkt := <-a.sendCh:
			a.conn.Write(pkt.b)

			// send twice: once for confirming with the queue,
			// again for timeout calculations
			for i := 0; i < 2; i++ {
				pkt.sent <- true
			}
		}
	}
}

func (a *AniDBUDP) recvLoop() {
	pkt := make(chan packet, 1)
	brk := make(chan bool)
	go func() {
		for {
			select {
			case <-brk:
				brk <- true
				return
			default:
				b, err := getPacket(a.conn, a.ecb)
				pkt <- packet{b: b, err: err}
			}
		}
	}()

	var pingTimer <-chan time.Time

	for {
		if a.pingTimer != nil {
			pingTimer = a.pingTimer.C
		}

		select {
		case <-a.breakRecv:
			brk <- true
			<-brk
			a.breakRecv <- true
			return
		case <-pingTimer:
			go func() {
				if a.KeepAliveInterval >= 30*time.Minute {
					if (<-a.Uptime()).Error() != nil {
						return
					}
				} else if (<-a.Ping()).Error() != nil {
					return
				}
				a.pingTimer.Reset(a.KeepAliveInterval)
			}()
		case p := <-pkt:
			b, err := p.b, p.err

			if err != nil && err != io.EOF && err != zlib.ErrChecksum {
				// can UDP recv even raise other errors?
				panic("UDP recv: " + err.Error())
			}

			if r := newGenericReply(b); r != nil {
				if a.pingTimer != nil {
					a.pingTimer.Reset(a.KeepAliveInterval)
				}

				if err == zlib.ErrChecksum {
					r.truncated = true
				}

				a.routerLock.RLock()
				if ch, ok := a.tagRouter[r.Tag()]; ok {

					log.Println("<<<", string(b))
					ch <- r
				} else {
					c := r.Code()
					if c >= 720 && c < 799 {
						// notices that need PUSHACK
						id := strings.Fields(r.Text())[0]
						a.send("PUSHACK", paramMap{"nid": id})

						a.Notifications <- r
					} else if c == 799 {
						// notice that doesn't need PUSHACK
						a.Notifications <- r
					} else if c == 270 {
						// PUSH enabled
						if a.pingTimer == nil {
							a.pingTimer = time.NewTimer(a.KeepAliveInterval)
						}
					} else if c == 370 {
						// PUSH disabled
						a.pingTimer = nil
					} else if c == 701 || c == 702 {
						// PUSHACK ACK, no need to route
					} else if c == 281 || c == 282 || c == 381 || c == 382 {
						// NOTIFYACK reply, ignore
					} else {
						// untagged error, broadcast to all
						log.Println("<!<", string(b))
						for _, ch := range a.tagRouter {
							ch <- r
						}
					}
				}
				a.routerLock.RUnlock()
			}
		}
	}
}
