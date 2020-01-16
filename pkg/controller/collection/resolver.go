package collection

import (
	"regexp"
	"strconv"

	"github.com/blang/semver"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	"gopkg.in/yaml.v2"
)

// ResolveIndex returns a structure representation of the yaml file represented by the index.
func ResolveIndex(repoConf kabanerov1alpha1.RepositoryConfig, pipelines []Pipelines, triggers []Trigger, imagePrefix string) (*Index, error) {
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

	processIndexPostRead(&index, pipelines, triggers, imagePrefix)

	index.URL = url

	return &index, nil
}

// Updates the loaded stack index structure for compliance with the current implementation.
func processIndexPostRead(index *Index, pipelines []Pipelines, triggers []Trigger, imagePrefix string) error {
	// Add common pipelines and image.
	for i, collection := range index.Collections {
		if len(collection.Pipelines) == 0 {
			collection.Pipelines = pipelines
		}

		if len(collection.Images) == 0 {
			version, err := semver.ParseTolerant(collection.Version)
			if err != nil {
				return err
			}

			image := imagePrefix + "/" + collection.Id + ":" + strconv.FormatUint(version.Major, 10) + "." + strconv.FormatUint(version.Minor, 10)
			collection.Images = append(collection.Images, Images{Id: collection.Id, Image: image})
		}

		index.Collections[i] = collection
	}

	// Add common triggers.
	if len(index.Triggers) == 0 {
		index.Triggers = triggers
	}

	return nil
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
