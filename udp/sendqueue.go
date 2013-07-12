package udpapi

import (
	"time"
)

type packet struct {
	/*...*/
	sent chan bool
}

type enqueuedPacket struct {
	packet
	queue chan packet
}

type sendQueueState struct {
	enqueue chan enqueuedPacket
}

var globalQueue sendQueueState

func init() {
	globalQueue = sendQueueState{
		enqueue: make(chan enqueuedPacket, 10),
	}
	go globalQueue.sendQueueDispatch()
}

const (
	throttleMinDuration = 2 * time.Second
	throttleMaxDuration = 4 * time.Second
	throttleIncFactor   = 1.1
	throttleDecFactor   = 0.9
	throttleDecInterval = 10 * time.Second
)

func sendPacket(p packet, c chan packet) {
	p.sent = make(chan bool, 2)
	globalQueue.enqueue <- enqueuedPacket{packet: p, queue: c}
}

func (gq *sendQueueState) sendQueueDispatch() {
	pkt := (*enqueuedPacket)(nil)
	queue := make([]enqueuedPacket, 0)

	nextTimer := time.NewTimer(0)
	decTimer := time.NewTimer(0)

	currentThrottle := throttleMinDuration

	for {
		if pkt == nil && len(queue) > 0 {
			pkt = &queue[0]
			queue = queue[1:]
		}

		nextCh := nextTimer.C
		decCh := decTimer.C

		if pkt == nil {
			nextCh = nil
		}

		select {
		case p := <-gq.enqueue:
			queue = append(queue, p)
		case <-nextCh:
			pkt.queue <- pkt.packet
			<-pkt.packet.sent

			pkt = nil

			currentThrottle = time.Duration(float64(currentThrottle) * throttleIncFactor)
			if currentThrottle > throttleMaxDuration {
				currentThrottle = throttleMaxDuration
			}
			nextTimer.Reset(currentThrottle)

			decTimer.Reset(throttleDecInterval)
		case <-decCh:
			currentThrottle = time.Duration(float64(currentThrottle) * throttleDecFactor)
			if currentThrottle < throttleMinDuration {
				currentThrottle = throttleMinDuration
			} else {
				decTimer.Reset(throttleDecInterval)
			}
		}
	}
}
