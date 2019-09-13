// +build dev

package config

import (
	"net/http"
	"path"
	"runtime"
)

var assets http.FileSystem = func() http.FileSystem {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		panic("Could not get the current caller")
	}
	filepath := path.Join(path.Dir(filename), "../../../config")
	fs := http.Dir(filepath)

	return fs
}()
