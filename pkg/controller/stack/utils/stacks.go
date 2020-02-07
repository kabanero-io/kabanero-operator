package utils

import (
	"fmt"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	"strings"
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

// Retrieves the repository part of input image. An error is returned if the image input is empty.
// If the image does not contain a tag, the  input value is returned.
func GetImageRepository(image string) (string, error) {
	if len(image) == 0 {
		return "", fmt.Errorf("The input image is empty.")
	}

	imageParts := strings.Split(image, ":")
	return imageParts[0], nil
}
