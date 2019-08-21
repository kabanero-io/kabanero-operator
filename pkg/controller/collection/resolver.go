package collection

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"strings"
	"regexp"
)

func ResolveIndex(url string) (*CollectionV1Index, error) {
//	if !strings.HasSuffix(url, "/index.yaml") {
//		url = url + "/index.yaml"
//	}
	
	// user may specify url to yaml file or directory
	matched, err := regexp.MatchString(`/([^/]+)[.]yaml$`, url) 
	if err != nil {
		return nil, err
	}
	if !matched {
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

// Return all resolved collections in the index matching the given name.
func SearchCollection(collectionName string, index *CollectionV1Index) ([]CollectionV1, error) {
	//Locate the desired collection in the index
	var collectionRefs []IndexedCollectionV1
	for _, collectionList := range index.Collections {
		for _, _collectionRef := range collectionList {
			if _collectionRef.Name == collectionName {
				collectionRefs = append(collectionRefs, _collectionRef)
			}
		}
	}

	if len(collectionRefs) == 0 {
		//The collection referenced in the Collection resource has no match in the index
		return nil, nil
	}

	var collections []CollectionV1
	for _, collectionRef := range collectionRefs {
		collection, err := ResolveCollection(collectionRef.CollectionUrls...)
		if err != nil {
			// TODO: somehow get this error back to the caller, but keep looking at other refs...
			return nil, err
		}

		if collection != nil {
			collections = append(collections, *collection)
		}
	}

	return collections, nil
}


// Return all resolved collections in the index matching the given name.
func SearchCollectionV2(collectionName string, index *CollectionV1Index) ([]IndexedCollectionV2, error) {
	//Locate the desired collection in the index
	var collectionRefs []IndexedCollectionV2
	
	for _, collectionRef := range index.CollectionsV2 {
		if collectionRef.Id == collectionName {
			collectionRefs = append(collectionRefs, collectionRef)
		}
	}

	if len(collectionRefs) == 0 {
		//The collection referenced in the Collection resource has no match in the index
		return nil, nil
	}

	return collectionRefs, nil
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
