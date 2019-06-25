package config

import (
	"net/http"
)

func Open(path string) (http.File, error) {
	file, err := assets.Open(path)
	if err != nil {
		return nil, err
	}

	return file, nil
}
