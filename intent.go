package anidb

import "sync"

type intentStruct struct {
	sync.Mutex
	chs []chan Cacheable
}

type intentMapStruct struct {
	sync.Mutex
	m map[string]*intentStruct
}

var intentMap = &intentMapStruct{
	m: map[string]*intentStruct{},
}

// Register a channel to be notified when the specified keys are notified.
//
// Cache checks should be done after registering intent, since it's possible to
// register Intent while a Notify is running, and the Notify is done after
// setting the cache.
func (m *intentMapStruct) Intent(ch chan Cacheable, keys ...cacheKey) bool {
	key := cachePath(keys...)

	m.Lock()
	s, ok := m.m[key]
	if !ok {
		s = &intentStruct{}
		m.m[key] = s
	}
	m.Unlock()

	s.Lock()
	s.chs = append(s.chs, ch)
	s.Unlock()

	return ok
}

// Notify all channels that are listening for the specified keys.
//
// Should be called after setting the cache.
func (m *intentMapStruct) Notify(v Cacheable, keys ...cacheKey) {
	key := cachePath(keys...)

	m.Lock()
	defer m.Unlock()
	s, ok := m.m[key]
	if !ok {
		return
	}

	s.Lock()
	defer s.Unlock()

	for _, ch := range s.chs {
		go func(c chan Cacheable) { c <- v }(ch)
	}

	delete(m.m, key)
}
