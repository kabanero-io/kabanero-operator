package utils

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	corev1 "k8s.io/api/core/v1"
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

// Custom filter method that allows the retrieval of a secret containing an annotation with a key of the
// form [string]-[number] (i.e kabanero.io/git-0), and a value that contains a hostname/domain (i.e. github.com).
// If there are multiple matching secrets, the annotation with the lexically lowest value key is used.
// If no secret matches annotation key, but there are secrets with an annotation value
// that matches the hostname/domain, the first one seen is used. If a secret could not be found, nil is returned.
func SecretAnnotationFilter(secretList *corev1.SecretList, filterStrings ...string) (*corev1.Secret, error) {
	var keyMatchingSecret *corev1.Secret = nil
	var noKeyMatchingSecret *corev1.Secret = nil
	kabKey := ""
	for _, secret := range secretList.Items {
		annotations := secret.GetAnnotations()
		for key, value := range annotations {
			matchedUrl, err := regexp.MatchString("^https?://", value)
			if err != nil {
				return nil, err
			}
			if matchedUrl {
				surl, err := url.Parse(value)
				if err != nil {
					fmt.Println("Unable to parse secret annotation value URL: ", surl, ". Secret: ", secret)
				}

				if surl.Hostname() == filterStrings[0] {
					if strings.HasPrefix(key, filterStrings[1]) {
						if len(kabKey) == 0 {
							kabKey = key
							keyMatchingSecret = secret.DeepCopy()
						} else {
							if kabKey > key {
								kabKey = key
								keyMatchingSecret = secret.DeepCopy()
							}
						}
					} else {
						// Save the first secret we see matching the hostname.
						if noKeyMatchingSecret == nil {
							noKeyMatchingSecret = secret.DeepCopy()
						}
					}
				}
			}
		}
	}

	if keyMatchingSecret != nil {
		return keyMatchingSecret, nil
	}
	if noKeyMatchingSecret != nil {
		return noKeyMatchingSecret, nil
	}

	return nil, nil
}
