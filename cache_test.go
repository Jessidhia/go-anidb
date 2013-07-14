package anidb

import (
	"encoding/gob"
	"os"
	"path"
	"reflect"
	"testing"
)

type stringifyVec struct {
	result []string
	data   []interface{}
}

func TestStringify(T *testing.T) {
	T.Parallel()

	vec := []stringifyVec{
		stringifyVec{[]string{"a"}, []interface{}{"a"}},
	}
	for i, v := range vec {
		str := stringify(v.data...)
		if !reflect.DeepEqual(v.result, str) {
			T.Errorf("Vector #%d: Expected %v, got %v", i+1, v.result, str)
		}
	}
}

type cachePathVec struct {
	path string
	data []interface{}
}

var testDir = path.Join(os.TempDir(), "testing", "anidb")

func init() { SetCacheDir(testDir) }

func TestCachePath(T *testing.T) {
	T.Parallel()

	vec := []cachePathVec{
		cachePathVec{path.Join(testDir, "a"), []interface{}{"a"}},
		cachePathVec{path.Join(testDir, "b", "c", "d"), []interface{}{"b", "c", "d"}},
	}
	for i, v := range vec {
		str := cachePath(v.data...)

		if v.path != str {
			T.Errorf("Vector #%d: Expected %v, got %v", i+1, v.path, str)
		}
	}
}

type testString string

func (_ testString) Touch()        {}
func (_ testString) IsStale() bool { return false }

func init() {
	gob.Register(testString(""))
}

func TestCacheRoundtrip(T *testing.T) {
	T.Parallel()

	test := testString("some string")
	_, err := cache.Set(test, "test", "string")
	if err != nil {
		T.Fatalf("Error storing: %v", err)
	}

	var t2 testString
	err = cache.Get(&t2, "test", "string")
	if err != nil {
		T.Errorf("Error reading: %v", err)
	}

	if test != t2 {
		T.Errorf("Expected %q, got %q", test, t2)
	}
}
