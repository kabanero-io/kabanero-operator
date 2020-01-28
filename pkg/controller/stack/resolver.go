package stack

import (
	"regexp"
	"strconv"

	"github.com/blang/semver"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	"gopkg.in/yaml.v2"
)

// ResolveIndex returns a structure representation of the yaml file represented by the index.
func ResolveIndex(repoConf kabanerov1alpha2.RepositoryConfig, pipelines []Pipelines, triggers []Trigger, imagePrefix string) (*Index, error) {
	url := repoConf.Https.Url

	// user may specify url to yaml file or directory
	matched, err := regexp.MatchString(`/([^/]+)[.]yaml$`, url)
	if err != nil {
		return nil, err
	}
	if !matched {
		url = url + "/index.yaml"
	}

	b, err := getFromCache(url, repoConf.Https.SkipCertVerification)
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
	for i, stack := range index.Stacks {
		if len(stack.Pipelines) == 0 {
			stack.Pipelines = pipelines
		}

		if len(stack.Images) == 0 {
			version, err := semver.ParseTolerant(stack.Version)
			if err != nil {
				return err
			}

			image := imagePrefix + "/" + stack.Id + ":" + strconv.FormatUint(version.Major, 10) + "." + strconv.FormatUint(version.Minor, 10)
			stack.Images = append(stack.Images, Images{Id: stack.Id, Image: image})
		}

		index.Stacks[i] = stack
	}

	// Add common triggers.
	if len(index.Triggers) == 0 {
		index.Triggers = triggers
	}

	return nil
}

// SearchStack returns all stacks in the index matching the given name.
func SearchStack(stackName string, index *Index) ([]Stack, error) {
	//Locate the desired stack in the index
	var stackRefs []Stack

	for _, stackRef := range index.Stacks {
		if stackRef.Id == stackName {
			stackRefs = append(stackRefs, stackRef)
		}
	}

	if len(stackRefs) == 0 {
		//The stack referenced in the Stack resource has no match in the index
		return nil, nil
	}

	return stackRefs, nil
}
