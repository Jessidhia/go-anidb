package anidb

type winFileLock struct{}

func lockFile(p string) fileLock {
	return &winFileLock{}
}

// empty implementations -- go-locking doesn't support windows
// windows also does file locking on its own
func (_ *winFileLock) Lock() error   { return nil }
func (_ *winFileLock) Unlock() error { return nil }
