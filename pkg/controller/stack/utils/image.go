package utils

import (
	"fmt"
	"strings"
)

const (
	defaultImageRegistry = "docker.io"
)

// Retrieves the repository part of input image that contains both a repo path and tag.
// An error is returned if the image input is empty.
//
// Reference (docker-specific, but should apply to others):
//   https://github.com/docker/distribution/blob/release/2.7/reference/reference.go
//   https://docs.docker.com/engine/reference/commandline/tag/
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
	repo := image
	tagIndex := strings.LastIndex(image, ":")
	slashIndex := strings.LastIndex(image, "/")
	if (tagIndex >= 0) && (slashIndex < tagIndex) {
		repo = image[0:tagIndex]
	}

	return repo, nil
}

// Retrieves the registry (domain) part of the input image. If a registry is not found, the default
// registry (docker.io) is returned.
// Reference: https://github.com/docker/distribution/blob/release/2.7/reference/reference.go
func GetImageRegistry(image string) (string, error) {
	// An image must have at least one path component preceded by a domain or followed by another
	// path component. If there is no slash, the image does not contain a domain.
	imageParts := strings.SplitN(image, "/", 2)
	if len(imageParts) != 2 {
		return defaultImageRegistry, nil
	}

	// The registry/domain can contain ':'. Validate that the string does not contain '_'. If it does, the
	// image is invalid.
	if strings.Contains(imageParts[0], ":") {
		if strings.Contains(imageParts[0], "_") {
			return "", fmt.Errorf("Invalid format: %v. Image: %v", imageParts[0], image)
		} else {
			return imageParts[0], nil
		}
	}

	// The registry/domain can contain '.'. If the string contains '_', the string is a path component.
	if strings.Contains(imageParts[0], ".") {
		if !strings.Contains(imageParts[0], "_") {
			return imageParts[0], nil
		}
	}

	// A registry was not recognized, return the default.
	return defaultImageRegistry, nil
}
