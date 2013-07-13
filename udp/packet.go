package udpapi

import (
	"bytes"
	"compress/zlib"
	"io"
	"io/ioutil"
	"net"
)

type packet struct {
	b    []byte
	err  error
	sent chan bool
}

func getPacket(conn *net.UDPConn, ecb *ecbState) (buf []byte, err error) {
	buf = make([]byte, 1500)
	n, err := conn.Read(buf)

	buf = ecb.Decrypt(buf[:n])

	if buf[0] == 0 && buf[1] == 0 {
		def, _ := zlib.NewReader(bytes.NewReader(buf[2:]))
		t, e := ioutil.ReadAll(def)
		def.Close()
		buf = t
		if e != nil && e != io.EOF {
			err = e
		}
	}
	return buf, err
}

func makePacket(buf []byte, ecb *ecbState) packet {
	return packet{b: ecb.Encrypt(buf)}
}
