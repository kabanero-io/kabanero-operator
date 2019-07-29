package collection

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"strings"
)

func ResolveIndex(url string) (*CollectionV1Index, error) {
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

func SearchCollection(collectionName string, index *CollectionV1Index) (*CollectionV1, error) {
	//Locate the desired collection in the index
	var collectionRef *IndexedCollectionV1
	for _, collectionList := range index.Collections {
		for _, _collectionRef := range collectionList {
			if _collectionRef.Name == collectionName {
				collectionRef = &_collectionRef
			}
		}
	}

	if collectionRef == nil {
		//The collection referenced in the Collection resource has no match in the index
		return nil, nil
	}

	collection, err := ResolveCollection(collectionRef.CollectionUrls...)
	if err != nil {
		return nil, err
	}

	return collection, nil
}

func ResolveCollection(urls ...string) (*CollectionV1, error) {
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
