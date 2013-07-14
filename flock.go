package anidb

type fileLock interface {
	Lock() error
	Unlock() error
}

// func lockFile(p path) fileLock
