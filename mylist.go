package anidb

import (
	"time"
)

type LID int64

type MyListState int

const (
	MyListStateUnknown = MyListState(iota)
	MyListStateHDD
	MyListStateCD
	MyListStateDeleted
)

type FileState int

const (
	FileStateOriginal = FileState(iota)
	FileStateCorrupted
	FileStateEdited

	FileStateOther = 100
)
const (
	FileStateSelfRip = FileState(10 + iota)
	FileStateDVD
	FileStateVHS
	FileStateTV
	FileStateTheaters
	FileStateStreamed
)

type MyListEntry struct {
	LID LID

	FID FID
	EID EID
	AID AID
	GID GID

	DateAdded   time.Time
	DateWatched time.Time

	State       FileState
	MyListState MyListState

	Storage string
	Source  string
	Other   string

	Cached time.Time
}
