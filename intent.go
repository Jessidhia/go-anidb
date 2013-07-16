package anidb

import (
	"github.com/Kovensky/go-fscache"
	"strings"
	"sync"
)

type notification interface{}

type intentStruct struct {
	sync.Mutex
	chs []chan notification
}

type intentMapStruct struct {
	sync.Mutex
	intentLock sync.Mutex // used by the Intent function

	m map[string]*intentStruct
}

var intentMap = &intentMapStruct{
	m: map[string]*intentStruct{},
}

func intentKey(key ...fscache.CacheKey) string {
	return strings.Join(fscache.Stringify(key...), "-")
}

// Register a channel to be notified when the specified keys are notified.
// Returns whether the caller was the first to register intent for the given
// keys.
//
// Cache checks should be done after registering intent, since it's possible to
// register Intent while a Notify is running, and the Notify is done after
// setting the cache.
func (m *intentMapStruct) Intent(ch chan notification, keys ...fscache.CacheKey) bool {
	key := intentKey(keys...)

	m.intentLock.Lock()
	defer m.intentLock.Unlock()

	m.Lock()
	s, ok := m.m[key]
	if !ok {
		s = &intentStruct{}
	}
	m.Unlock()

	s.Lock()
	s.chs = append(s.chs, ch)
	s.Unlock()

	m.Lock()
	// key might have been deleted while only the struct itself was locked -- recheck
	_, ok = m.m[key]
	m.m[key] = s
	m.Unlock()

	return ok
}

// Locks the requested keys and return the locked intentStruct.
//
// The intentStruct can be directly unlocked, or given to Free to also
// remove it from the intent map.
func (m *intentMapStruct) LockIntent(keys ...fscache.CacheKey) *intentStruct {
	m.Lock()
	defer m.Unlock()

	return m._lockIntent(keys...)
}

func (m *intentMapStruct) _lockIntent(keys ...fscache.CacheKey) *intentStruct {
	s, ok := m.m[intentKey(keys...)]
	if !ok {
		return nil
	}

	s.Lock()
	return s
}

// Removes the given intent from the intent map and unlocks the intentStruct.
func (m *intentMapStruct) Free(is *intentStruct, keys ...fscache.CacheKey) {
	m.Lock()
	defer m.Unlock()

	m._free(is, keys...)
}

func (m *intentMapStruct) _free(is *intentStruct, keys ...fscache.CacheKey) {
	// deletes the key before unlocking, Intent needs to recheck key status
	delete(m.m, intentKey(keys...))
	// better than unlocking then deleting -- could delete a "brand new" entry
	is.Unlock()
}

// Notifies and closes all channels that are listening for the specified keys;
// also removes them from the intent map.
//
// Should be called after setting the cache.
func (m *intentMapStruct) NotifyClose(v notification, keys ...fscache.CacheKey) {
	m.Lock()
	defer m.Unlock()

	is := m._lockIntent(keys...)
	defer m._free(is, keys...)

	is.NotifyClose(v)
}

// Closes all channels that are listening for the specified keys
// and removes them from the intent map.
func (m *intentMapStruct) Close(keys ...fscache.CacheKey) {
	m.Lock()
	defer m.Unlock()

	is := m._lockIntent(keys...)
	defer m._free(is, keys...)

	is.Close()
}

// Notifies all channels that are listening for the specified keys,
// but doesn't close or remove them from the intent map.
func (m *intentMapStruct) Notify(v notification, keys ...fscache.CacheKey) {
	m.Lock()
	defer m.Unlock()

	is := m._lockIntent(keys...)
	defer is.Unlock()

	is.Notify(v)
}

// NOTE: does not lock the stuct
func (s *intentStruct) Notify(v notification) {
	for _, ch := range s.chs {
		ch <- v
	}
}

// NOTE: does not lock the struct
func (s *intentStruct) Close() {
	for _, ch := range s.chs {
		close(ch)
	}
	s.chs = nil
}

// NOTE: does not lock the struct
func (s *intentStruct) NotifyClose(v notification) {
	for _, ch := range s.chs {
		ch <- v
		close(ch)
	}
	s.chs = nil
}
