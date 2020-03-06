package utils

import (
	"fmt"
	"strings"

	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
)

// Removes the tag portion of all images associated with the input stack version.
func RemoveTagFromStackImages(stack *kabanerov1alpha2.StackVersion, stackName string) error {
	for j, image := range stack.Images {
		repo, err := GetImageRepository(image.Image)
		if err != nil {
			return fmt.Errorf("Unable to process image %v associated with stack %v %v. Error: %v", image.Image, stackName, stack.Version, err)
		}
		stack.Images[j].Image = repo
	}

	return nil
}

// Retrieves the repository part of input image that contains both a repo path and tag.
// An error is returned if the image input is empty.
func GetImageRepository(image string) (string, error) {
	if len(image) == 0 {
		return "", fmt.Errorf("The input image is empty.")
	}

	// A tag name must be valid ASCII and may contain lowercase and uppercase
	// letters, digits, underscores, periods and dashes.  A tag name may not
	// start with a period or a dash, and may contain a maximum of 128
	// characters.
	//
	// The tag cannot contain a slash.  So, if we find a slash past the last
	// colon, what we found is not a tag and we should leave it alone.
	//
	// Additionally, the components of a repository name cannot contain
	// a colon, with the exception of the first component which is actually
	// a hostname.  So if the repository contains a colon which is not the
	// tag separator, it must occur in the first component, and there must
	// also be a second component after the first (implying there is a
	// slash between the components).
	//
	// Reference (docker-specific, but should apply to others):
	//   https://github.com/docker/distribution/blob/release/2.7/reference/reference.go
	//   https://docs.docker.com/engine/reference/commandline/tag/
	
	repo := image
	tagIndex := strings.LastIndex(image, ":")
	slashIndex := strings.LastIndex(image, "/")
	if (tagIndex >= 0) && (slashIndex < tagIndex) {
		repo = image[0:tagIndex]
	}

	return repo, nil
}
