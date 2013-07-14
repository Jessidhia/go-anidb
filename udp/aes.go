package udpapi

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
)

// Yes, AniDB works in ECB mode
type ecbState struct {
	cipher.Block
}

func newECBState(udpKey string, salt []byte) *ecbState {
	ecb := &ecbState{}
	ecb.Init(udpKey, salt)
	return ecb
}

func (ecb *ecbState) Init(udpKey string, salt []byte) {
	h := md5.New()
	h.Write([]byte(udpKey))
	h.Write(salt)

	key := h.Sum(nil)

	b, err := aes.NewCipher(key)
	ecb.Block = b
	if err != nil {
		panic(err)
	}
}

func (ecb *ecbState) BlockSize() int {
	return aes.BlockSize
}

func (ecb *ecbState) Encrypt(p []byte) (c []byte) {
	if ecb == nil {
		return p
	}

	padded := pkcs7Pad(p, aes.BlockSize)
	c = make([]byte, len(padded))

	for i := 0; i < len(padded); i += aes.BlockSize {
		ecb.Block.Encrypt(c[i:i+aes.BlockSize], padded[i:i+aes.BlockSize])
	}
	return c
}

func (ecb *ecbState) Decrypt(c []byte) []byte {
	if ecb == nil {
		return c
	}

	p := make([]byte, len(c))
	for i := 0; i < len(c); i += aes.BlockSize {
		ecb.Block.Decrypt(p[i:i+aes.BlockSize], c[i:i+aes.BlockSize])
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
