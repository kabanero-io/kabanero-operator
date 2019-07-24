package collection

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"strings"
)

func resolveIndex(url string) (*CollectionV1Index, error) {
	if !strings.HasSuffix(url, "/index.yaml") {
		url = url + "/index.yaml"
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("Could not resolve the index: %v", url))
	}
	r := resp.Body
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var index CollectionV1Index
	err = yaml.Unmarshal(b, &index)
	if err != nil {
		return nil, err
	}

	return &index, nil
}

func resolveCollection(urls ...string) (*CollectionV1, error) {
	for _, url := range urls {
		if strings.HasSuffix(url, "tar.gz") {
			panic("No implementation for collection archives")
		} else if !strings.HasSuffix(url, "collection.yaml") {
			//Add collection.yaml to path
			if strings.HasSuffix(url, "/") {
				url = url + "collection.yaml"
			} else {
				url = url + "/collection.yaml"
			}
		}

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		r := resp.Body
		b, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}

		var manifest CollectionV1Manifest
		err = yaml.Unmarshal(b, &manifest)
		if err != nil {
			return nil, err
		}

		collection := &CollectionV1{
			Manifest: manifest,
		}

		return collection, nil
	}

	return nil, nil
}
