package anidb

import (
	"github.com/Kovensky/go-anidb/udp"
	"github.com/Kovensky/go-fscache"
	"strconv"
	"strings"
	"sync"
	"time"
)

type UID int

func (adb *AniDB) GetCurrentUser() <-chan *User {
	ch := make(chan *User, 1)

	if adb.udp.credentials == nil {
		ch <- nil
		close(ch)
		return ch
	}

	return adb.GetUserByName(decrypt(adb.udp.credentials.username))
}

// This is an (almost) entirely local representation.
func (adb *AniDB) GetUserByID(uid UID) <-chan *User {
	key := []fscache.CacheKey{"user", uid}
	ch := make(chan *User, 1)

	if uid < 1 {
		ch <- nil
		close(ch)
		return ch
	}

	ic := make(chan notification, 1)
	go func() { ch <- (<-ic).(*User); close(ch) }()
	if intentMap.Intent(ic, key...) {
		return ch
	}

	if !Cache.IsValid(InvalidKeyCacheDuration, key...) {
		intentMap.NotifyClose((*User)(nil), key...)
		return ch
	}

	go func() {
		var user *User
		if CacheGet(&user, key...) == nil {
			intentMap.NotifyClose(user, key...)
			return
		}
		<-adb.GetUserName(uid)

		CacheGet(&user, key...)
		intentMap.NotifyClose(user)
	}()
	return ch
}

func (adb *AniDB) GetUserByName(username string) <-chan *User {
	ch := make(chan *User, 1)

	if username == "" {
		ch <- nil
		close(ch)
		return ch
	}

	go func() {
		ch <- <-adb.GetUserByID(<-adb.GetUserUID(username))
		close(ch)
	}()
	return ch
}

func (adb *AniDB) GetUserUID(username string) <-chan UID {
	key := []fscache.CacheKey{"user", "by-name", username}
	ch := make(chan UID, 1)

	if username == "" {
		ch <- 0
		close(ch)
		return ch
	}

	ic := make(chan notification, 1)
	go func() { ch <- (<-ic).(UID); close(ch) }()
	if intentMap.Intent(ic, key...) {
		return ch
	}

	if !Cache.IsValid(InvalidKeyCacheDuration, key...) {
		intentMap.NotifyClose((UID)(0), key...)
		return ch
	}

	uid := UID(0)
	switch ts, err := Cache.Get(&uid, key...); {
	case err == nil && time.Now().Sub(ts) < UIDCacheDuration:
		intentMap.NotifyClose(uid, key...)
		return ch
	}

	go func() {
		reply := <-adb.udp.SendRecv("USER",
			paramMap{"user": username})

		switch reply.Code() {
		case 295:
			uid, _ = parseUserReply(reply) // caches
		case 394:
			Cache.SetInvalid(key...)
		}

		intentMap.NotifyClose(uid, key...)
	}()
	return ch
}

func (adb *AniDB) GetUserName(uid UID) <-chan string {
	key := []fscache.CacheKey{"user", "by-uid", uid}
	ch := make(chan string, 1)

	if uid < 1 {
		ch <- ""
		close(ch)
		return ch
	}

	ic := make(chan notification, 1)
	go func() { ch <- (<-ic).(string); close(ch) }()
	if intentMap.Intent(ic, key...) {
		return ch
	}

	if !Cache.IsValid(InvalidKeyCacheDuration, key...) {
		intentMap.NotifyClose("", key...)
		return ch
	}

	name := ""
	switch ts, err := Cache.Get(&name, key...); {
	case err == nil && time.Now().Sub(ts) < UIDCacheDuration:
		intentMap.NotifyClose(name, key...)
		return ch
	}

	go func() {
		reply := <-adb.udp.SendRecv("USER",
			paramMap{"uid": uid})

		switch reply.Code() {
		case 295:
			_, name = parseUserReply(reply) // caches
		case 394:
			Cache.SetInvalid(key...)
		}

		intentMap.NotifyClose(name, key...)
	}()
	return ch
}

var userReplyMutex sync.Mutex

func parseUserReply(reply udpapi.APIReply) (UID, string) {
	userReplyMutex.Lock()
	defer userReplyMutex.Unlock()

	if reply.Error() == nil {
		parts := strings.Split(reply.Lines()[1], "|")
		id, _ := strconv.ParseInt(parts[0], 10, 32)

		CacheSet(UID(id), "user", "by-name", parts[1])
		CacheSet(parts[1], "user", "by-uid", id)

		if _, err := Cache.Stat("user", id); err != nil {
			CacheSet(&User{
				UID:      UID(id),
				Username: parts[1],
			}, "user", id)
		}

		return UID(id), parts[1]
	}
	return 0, ""
}
