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

	repo := image
	tagIndex := strings.LastIndex(image, ":")
	if tagIndex >= 0 {
		repo = image[0:tagIndex]
	}

	return repo, nil
}
