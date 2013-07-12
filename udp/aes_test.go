package udp

import (
	"testing"
)

func TestPKCS7(T *testing.T) {
	blockSize := byte(4)
	vec := [][2]string{
		[2]string{"testing", "testing\x01"},
		[2]string{"byte", "byte\x04\x04\x04\x04"},
		[2]string{"stuff", "stuff\x03\x03\x03"},
	}

	for i, v := range vec {
		p := string(pkcs7Pad([]byte(v[0]), blockSize))
		if p != v[1] {
			T.Errorf("Vector #%d: expected %q, got %q", i, v[1], p)
		}
		u := string(pkcs7Unpad([]byte(p), blockSize))
		if u != v[0] {
			T.Errorf("Vector #%d: expected %q, got %q", i, v[0], u)
		}
	}
}
