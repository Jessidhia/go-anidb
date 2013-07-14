package anidb

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"runtime"
)

// We still have the key and IV somewhere in memory...
// but it's better than plaintext.
type credentials struct {
	username []byte
	password []byte
	udpKey   []byte
}

func (c *credentials) shred() {
	if c != nil {
		io.ReadFull(rand.Reader, c.username)
		io.ReadFull(rand.Reader, c.password)
		io.ReadFull(rand.Reader, c.udpKey)
		c.username = nil
		c.password = nil
		c.udpKey = nil
	}
}

// Randomly generated on every execution
var aesKey []byte

func init() {
	aesKey = make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, aesKey); err != nil {
		panic(err)
	}
}

func crypt(plaintext string) []byte {
	p := []byte(plaintext)

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		panic(err)
	}

	ciphertext := make([]byte, len(p)+aes.BlockSize)
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}

	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], p)

	return ciphertext
}

func decrypt(ciphertext []byte) string {
	if len(ciphertext) <= aes.BlockSize {
		return ""
	}
	p := make([]byte, len(ciphertext)-aes.BlockSize)

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		panic(err)
	}

	stream := cipher.NewCTR(block, ciphertext[:aes.BlockSize])
	stream.XORKeyStream(p, ciphertext[aes.BlockSize:])

	return string(p)
}

func newCredentials(username, password, udpKey string) *credentials {
	return &credentials{
		username: crypt(username),
		password: crypt(password),
		udpKey:   crypt(udpKey),
	}
}

func (udp *udpWrap) ReAuth() error {
	if c := udp.credentials; c != nil {
		defer runtime.GC() // any better way to clean the plaintexts?

		udp.connected = true
		return udp.Auth(
			decrypt(c.username),
			decrypt(c.password),
			decrypt(c.udpKey))
	}
	return errors.New("No credentials stored")
}

// Saves the used credentials in the AniDB struct, to allow automatic
// re-authentication when needed; they are (properly) encrypted with a key that's
// uniquely generated every time the module is initialized.
func (adb *AniDB) SetCredentials(username, password, udpKey string) {
	adb.udp.credentials.shred()
	adb.udp.credentials = newCredentials(username, password, udpKey)
}

// Authenticates to anidb's UDP API and, on success, stores the credentials using
// SetCredentials. If udpKey is not "", the communication with the server
// will be encrypted, but in the VERY weak ECB mode.
func (adb *AniDB) Auth(username, password, udpKey string) (err error) {
	defer runtime.GC() // any better way to clean the plaintexts?

	if err = adb.udp.Auth(username, password, udpKey); err == nil {
		adb.udp.connected = true
		adb.SetCredentials(username, password, udpKey)
	}
	return
}

// Logs the user out and removes the credentials from the AniDB struct.
func (adb *AniDB) Logout() error {
	if adb.udp.connected {
		adb.udp.credentials.shred()
		adb.udp.credentials = nil

		adb.udp.connected = false
		return adb.udp.Logout()
	}
	return nil
}
