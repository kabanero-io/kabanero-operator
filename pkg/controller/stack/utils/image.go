package utils

import (
	reference "github.com/docker/distribution/reference"
)


// Retrieves the repository part of input image that contains both a repo path and tag.
// An error is returned if the image input is empty.
//
// Reference (docker-specific, but should apply to others):
//   https://github.com/docker/distribution/blob/release/2.7/reference/reference.go
//   https://docs.docker.com/engine/reference/commandline/tag/
func GetImageRepository(image string) (string, error) {

	ref, err := reference.ParseAnyReference(image)
	if err != nil {
		return "", err
	}
	named, err := reference.ParseNormalizedNamed(ref.String())
	if err != nil {
		return "", err
	}
	repo := named.Name()

	return repo, nil
}


// Retrieves the registry (domain) part of the input image. If a registry is not found, the default
// registry (docker.io) is returned.
// Reference: https://github.com/docker/distribution/blob/release/2.7/reference/reference.go
func GetImageRegistry(image string) (string, error) {

	ref, err := reference.ParseAnyReference(image)
	if err != nil {
		return "", err
	}
	named, err := reference.ParseNormalizedNamed(ref.String())
	if err != nil {
		return "", err
	}
	domain := reference.Domain(named)

	return domain, nil
}
