// +build !windows

package anidb

import "github.com/tgulacsi/go-locking"

type flockLock struct {
	locking.FLock
}

func lockFile(p string) fileLock {
	flock, err := locking.NewFLock(p)
	if err == nil {
		return &flockLock{FLock: flock}
	}
	return nil
}

func (fl *flockLock) Lock() error {
	if fl != nil {
		return fl.FLock.Lock()
	}
	return nil
}

func (fl *flockLock) Unlock() error {
	if fl != nil {
		return fl.FLock.Unlock()
	}
	return nil
}
