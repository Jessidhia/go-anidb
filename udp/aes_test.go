package udpapi

import (
	"math/rand"
	"reflect"
	"testing"
)

func TestECB(T *testing.T) {
	T.Parallel()

	rnd := rand.New(rand.NewSource(31415))

	salt := make([]byte, rnd.Intn(64))
	for i := range salt {
		salt[i] = byte(rnd.Intn(255))
	}
	plain := make([]byte, 1200+rnd.Intn(200))
	for i := range plain {
		plain[i] = byte(rnd.Intn(255))
	}

	ecbState := newECBState("agaa", salt)

	T.Log("Length of plaintext:", len(plain))
	cipher := ecbState.Encrypt(plain)
	T.Log("Length of ciphertext:", len(cipher))
	plain2 := ecbState.Decrypt(cipher)
	T.Log("Length of roundtrip plaintext:", len(plain2))

	if !reflect.DeepEqual(plain, plain2) {
		T.Error("Encoding roundtrip result doesn't match plaintext")
	}
}

func TestPKCS7(T *testing.T) {
	T.Parallel()

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
