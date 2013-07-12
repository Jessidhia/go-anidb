package udpapi

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
)

// Yes, AniDB works in ECB mode
type ecbState struct {
	udpKey string
	aes    cipher.Block
}

func newECBState(udpKey string, salt []byte) *ecbState {
	ecb := &ecbState{udpKey: udpKey}
	ecb.Init(salt)
	return ecb
}

func (ecb *ecbState) Init(salt []byte) {
	h := md5.New()
	h.Write([]byte(ecb.udpKey))
	h.Write(salt)

	key := h.Sum(nil)

	ecb.aes, _ = aes.NewCipher(key)
}

func (ecb *ecbState) BlockSize() int {
	return aes.BlockSize
}

func (ecb *ecbState) Encrypt(p []byte) (c []byte) {
	if ecb == nil {
		return p
	}

	padded := pkcs7Pad(p, aes.BlockSize)
	c = make([]byte, 0, len(padded))

	for i := 0; i < len(padded); i += aes.BlockSize {
		ecb.aes.Encrypt(c[i:i+aes.BlockSize], padded[i:i+aes.BlockSize])
	}
	return c
}

func (ecb *ecbState) Decrypt(c []byte) (p []byte) {
	if ecb == nil {
		return c
	}

	for i := 0; i < len(c); i += ecb.aes.BlockSize() {
		ecb.aes.Decrypt(p[i:], c[i:])
	}
	return pkcs7Unpad(p, aes.BlockSize)
}

// examples for a blocksize of 4
// "almost1\x1"
// "bytes\x3\x3\x3"
// "byte\x4\x4\x4\x4"
func pkcs7Pad(b []byte, blockSize byte) (padded []byte) {
	ps := int(blockSize) - len(b)%int(blockSize)
	padded = make([]byte, 0, len(b)+ps)
	padded = append(padded, b...)

	for i := 0; i < ps; i++ {
		padded = append(padded, byte(ps))
	}
	return padded
}

func pkcs7Unpad(b []byte, blockSize byte) (unpadded []byte) {
	ps := b[len(b)-1]
	if ps > blockSize {
		return b
	}
	padding := b[len(b)-int(ps):]
	for _, pb := range padding {
		if pb != ps {
			return b
		}
	}
	return b[:len(b)-int(ps)]
}
