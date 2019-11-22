package collection

import (
	"regexp"

	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	"gopkg.in/yaml.v2"
)

// ResolveIndex returns a structure representation of the yaml file represented by the index.
func ResolveIndex(repoConf kabanerov1alpha1.RepositoryConfig) (*Index, error) {
	url := repoConf.Url

	// user may specify url to yaml file or directory
	matched, err := regexp.MatchString(`/([^/]+)[.]yaml$`, url)
	if err != nil {
		return nil, err
	}
	if !matched {
		url = url + "/index.yaml"
	}

	b, err := getFromCache(url, repoConf.SkipCertVerification)
	if err != nil {
		return nil, err
	}

	var index Index
	err = yaml.Unmarshal(b, &index)
	if err != nil {
		return nil, err
	}
	index.URL = url

	return &index, nil
}

// SearchCollection returns all collections in the index matching the given name.
func SearchCollection(collectionName string, index *Index) ([]Collection, error) {
	//Locate the desired collection in the index
	var collectionRefs []Collection

	for _, collectionRef := range index.Collections {
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
