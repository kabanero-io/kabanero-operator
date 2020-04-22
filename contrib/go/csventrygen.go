package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"

	"github.com/coreos/go-semver/semver"
	"github.com/kabanero-io/kabanero-operator/pkg/versioning"
	"gopkg.in/yaml.v2"
)

// Components from versions.yaml to include in CSV spec.relatedImages.
var components []string = []string{"cli-services", "landing", "events", "codeready-workspaces"}

// Builds the CSV spec.relatedImages content.
// Arguments:
// 1: true/false. When set to true only the most current component version is included in
//    the list of relatedImages. If false, all component versions (condifg/versions.yaml) with a version value
//    less or equal to the default kabanero version are included in the list of relatedImages.
func main() {
	currentReleaseOnly, err := strconv.ParseBool(os.Args[1])
	versionsRIList, defaultKabaneroRevision, err := getRelatedVersionsContent(currentReleaseOnly)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get related image content. Error: %v\n", err)
		os.Exit(1)
	}

	// Find Spec.relatedImages
	csvRIList, err := getCSVRelatedImagesContent(defaultKabaneroRevision)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while attempting to read the relatedImages entry from CSV. Error: %v\n", err)
		os.Exit(1)
	}

	// If Spec.relatedImages is empty or not present in the CSV file, write the list generated from versions.yaml to a file.
	if len(csvRIList) == 0 {
		err := printRelatedImgesYaml(versionsRIList)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing CSV related images entries to a file. Error: %v\n", err)
			os.Exit(1)
		}
	}

	// Spec.relatedImages content is present in the CSV. Merge the list generated from versions.yaml and the retrieved CSV.
	mergedRIList, err := mergeCSVRelatedImagesEntry(versionsRIList, csvRIList)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error merging related images entries derived from versions.yaml and existing CSV file. Error: %v\n", err)
		os.Exit(1)
	}

	// Write the list of generated objects to a file. Note that a better approach would be to update the CSV file here directly.
	// However, scalar values contained in the CSV are mangled by the yaml APIs. As a workaround, the relatedImages yaml entry is
	// written to a file to be handled by the associated script.
	err = printRelatedImgesYaml(mergedRIList)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing CSV related images entries to a file. Error: %v\n", err)
		os.Exit(1)
	}
}

// Returns an identifier entry associated with the identifier name input.
func getIdentifierValue(identifiers map[string]interface{}, identifierName string) (string, error) {
	value := ""
	valueI, found := identifiers[identifierName]
	if found {
		v, ok := valueI.(string)
		if !ok {
			return "", fmt.Errorf(fmt.Sprintf("Unable to cast %v to string", valueI))
		}
		value = v
	}
	return value, nil
}

// Builds a related images object entry based on versions.yaml content.
// Name: [componentName]-[componentSoftwareVersion]
// Image: [repo]@sha256:[digest]
// Note temporarily support tags, as that is what is currently supported by the Kabanero Operator.
func buildRelatedImagesEntry(identifiers map[string]interface{}, componentName string, compSoftVersion string) (map[interface{}]interface{}, error) {
	if identifiers == nil {
		return nil, nil
	}

	var repo, tag, digest string
	if componentName == "codeready-workspaces" {
		r, err := getIdentifierValue(identifiers, "devfile-reg-repository")
		if err != nil {
			return nil, err
		}
		repo = r
		tag, err = getIdentifierValue(identifiers, "devfile-reg-tag")
		if err != nil {
			return nil, err
		}
		digest, err = getIdentifierValue(identifiers, "devfile-reg-digest")
		if err != nil {
			return nil, err
		}
	} else {
		r, err := getIdentifierValue(identifiers, "repository")
		if err != nil {
			return nil, err
		}
		repo = r
		tag, err = getIdentifierValue(identifiers, "tag")
		if err != nil {
			return nil, err
		}
		digest, err = getIdentifierValue(identifiers, "digest")
		if err != nil {
			return nil, err
		}

	}

	tmp := make(map[interface{}]interface{})
	tmp["name"] = componentName + "-" + compSoftVersion

	if len(digest) != 0 {
		tmp["image"] = repo + "@sha256:" + digest
	} else if len(tag) != 0 {
		tmp["image"] = repo + ":" + tag
	}

	return tmp, nil
}

// Builds a list related image objects from config/versions.yaml. The versions included are governed
// the default kabanero revision version and the currentReleaseOnly input. When this input is set to true, only the most
// current component version is included in the list of relatedImages. If false, all component versions
// with a version value less or equal to the default kabanero revision version are included in the list of relatedImages.
func getRelatedVersionsContent(currentReleaseOnly bool) ([]map[interface{}]interface{}, string, error) {
	versionsRIList := []map[interface{}]interface{}{}

	v := versioning.Data
	defaultKabaneroRevision := v.DefaultKabaneroRevision
	for _, revision := range v.KabaneroRevisions {
		// Process only software revisions at the specified default kabanero version.
		if defaultKabaneroRevision != revision.Version {
			continue
		}

		// Iterator over all component associated with the kabanero version we are interested in.
		revVersions := revision.RelatedVersions
		for _, componentName := range components {
			// Find the component version for the particular kabanero version.
			compMaxVersion, found := revVersions[componentName]
			if !found {
				err := fmt.Errorf(fmt.Sprintf("Revision data for component %v was not found. Revision versions map: %v", componentName, revVersions))
				return nil, "", err
			}

			compMaxVersionSemver, err := semver.NewVersion(compMaxVersion)
			if err != nil {
				err := fmt.Errorf(fmt.Sprintf("Unable to convert %v to semver object. Error: %v", compMaxVersion, err))
				return nil, "", err
			}

			// Iterate over all of the componet versions.
			listOfSoftRevs := revision.Document.RelatedSoftwareRevisions
			compSoftRevs, found := listOfSoftRevs[componentName]
			for _, compSoftRev := range compSoftRevs {
				compSoftVersion, err := semver.NewVersion(compSoftRev.Version)
				if err != nil {
					err := fmt.Errorf(fmt.Sprintf("Unable to convert %v to semver object. Error: %v", compSoftRev.Version, err))
					return nil, "", err
				}

				if currentReleaseOnly {
					// Build related image entry for the most recent release associated with the component.
					if compSoftVersion.Compare(*compMaxVersionSemver) == 0 {
						riEntry, err := buildRelatedImagesEntry(compSoftRev.Identifiers, componentName, compSoftRev.Version)
						if err != nil {
							return nil, "", err
						}
						if riEntry != nil {
							versionsRIList = append(versionsRIList, riEntry)
						}
					}
				} else if compSoftVersion.Compare(*compMaxVersionSemver) <= 0 {
					// Build related image entries for the most recent release and prior versions associated with the component.
					riEntry, err := buildRelatedImagesEntry(compSoftRev.Identifiers, componentName, compSoftRev.Version)
					if err != nil {
						return nil, "", err
					}
					if riEntry != nil {
						versionsRIList = append(versionsRIList, riEntry)
					}
				}
			}
		}
	}

	return versionsRIList, defaultKabaneroRevision, nil
}

// Retrieves the object list of related image entries currently present in the Kabanero CSV file.
func getCSVRelatedImagesContent(defaultKabaneroRevision string) ([]map[interface{}]interface{}, error) {
	csvFilePath := "registry/manifests/kabanero-operator/" + defaultKabaneroRevision + "/kabanero-operator.v" + defaultKabaneroRevision + ".clusterserviceversion.yaml"
	csvBytes, err := ioutil.ReadFile(csvFilePath)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("Unable to read CSV file %v. Error: %v", csvFilePath, err))
	}

	csv := make(map[string]interface{})
	err = yaml.Unmarshal(csvBytes, &csv)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("Unable to unmarshall CSV file %v. Error: %v", csvFilePath, err))
	}

	csvSpecI, found := csv["spec"]
	if !found {
		return nil, fmt.Errorf(fmt.Sprintf("Invalid CSV: %v. No spec entry found. Error: %v", csvFilePath, err))
	}

	var relatedImagesEntry []map[interface{}]interface{}
	csvSpec, ok := csvSpecI.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf(fmt.Sprintf("Unable to cast %v to map[interface{}]interface{}", csvSpecI))
	}

	specRiI, found := csvSpec["relatedImages"]
	if found {
		relatedImagesEntryI, ok := specRiI.([]interface{})
		if !ok {
			return nil, fmt.Errorf(fmt.Sprintf("Unable to cast %v to []interface{}", specRiI))
		}

		for _, entryI := range relatedImagesEntryI {
			entry, ok := entryI.(map[interface{}]interface{})
			if !ok {
				return nil, fmt.Errorf(fmt.Sprintf("Unable to cast %v to map[interface{}]interface{}.", entryI))
			}
			relatedImagesEntry = append(relatedImagesEntry, entry)
		}
	}

	return relatedImagesEntry, nil
}

// Merges the list of related image objects currently present in the CSV and the current one build from versions.yaml.
// Entries computed from versions.yaml take precedence.
func mergeCSVRelatedImagesEntry(versionsRIList []map[interface{}]interface{}, csvRIList []map[interface{}]interface{}) ([]map[interface{}]interface{}, error) {
	nameRegex := regexp.MustCompile(`-\d+.\d+.\d+`)
	versionRegex := regexp.MustCompile(`\d+.\d+.\d+`)
	mergedRIList := versionsRIList

	for i, csvRIMap := range csvRIList {
		nNameFound := false

		for csvRIIKeyI, csvRIIValueI := range csvRIMap {
			csvRIIKeyName, ok := csvRIIKeyI.(string)
			if !ok {
				return nil, fmt.Errorf(fmt.Sprintf("Unable to cast %v to string", csvRIIKeyI))
			}

			if csvRIIKeyName == "name" {
				csvRIIValueName, ok := csvRIIValueI.(string)
				if !ok {
					return nil, fmt.Errorf(fmt.Sprintf("Unable to cast %v to string", csvRIIValueI))
				}
				csvRIINameSlice := nameRegex.Split(csvRIIValueName, 2)
				csvRIICompName := csvRIINameSlice[0]
				csvRIICompVersion := versionRegex.FindString(csvRIIValueName)

				for _, versionsRI := range versionsRIList {
					for versionsRIIKey, versionsRIIValue := range versionsRI {
						versionsRIIKeyName, ok := versionsRIIKey.(string)
						if !ok {
							return nil, fmt.Errorf(fmt.Sprintf("Unable to cast %v to string", versionsRIIKey))
						}

						if ok && versionsRIIKeyName == "name" {
							versionsRIIValueName := versionsRIIValue.(string)
							if !ok {
								return nil, fmt.Errorf(fmt.Sprintf("Unable to cast %v to string", versionsRIIValue))
							}
							versionsRIINameSlice := nameRegex.Split(versionsRIIValueName, 2)
							versionsRIICompName := versionsRIINameSlice[0]
							versionsRIICompVersion := versionRegex.FindString(versionsRIIValueName)

							// The following check allows release candidate entries to be overriden by release version entries.
							if (csvRIICompName == versionsRIICompName) && (csvRIICompVersion == versionsRIICompVersion) {
								nNameFound = true
								break
							}
						}
					}
					if nNameFound {
						break
					}
				}
			}
			if nNameFound {
				break
			}
		}

		if !nNameFound {
			mergedRIList = append(mergedRIList, csvRIList[i])
		}
	}

	return mergedRIList, nil
}

// Prints the input data into a file.
func printRelatedImgesYaml(data []map[interface{}]interface{}) error {
	riEntry := make(map[string][]map[interface{}]interface{})
	riEntry["relatedImages"] = data

	// Write it to a file.
	rImgYamlPath := "contrib/go/csvRelatedImages.yaml"
	rImgBytes, err := yaml.Marshal(riEntry)
	err = ioutil.WriteFile(rImgYamlPath, rImgBytes, 0644)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Unable to write CSV bytes to file %v. Error: %v", rImgYamlPath, err))
	}
	return nil
}
