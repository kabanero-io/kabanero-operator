package utils

import (
	"fmt"
	"testing"
)

// Tests that GetImageRepository removes the tag from the input image.
// docker/distribution/reference will prefix hostname when absent
func TestGetImageRepository(t *testing.T) {
	// Test external repository:tag
	image := "kabanero/kabanero-image:1.2.3"
	repo, err := GetImageRepository(image)
	if err != nil {
		t.Fatal(fmt.Sprintf("Unexpected error while removing tag from image: %v. Error: ", err))
	}
	expectedRepo := "docker.io/kabanero/kabanero-image"
	if repo != expectedRepo {
		t.Fatal(fmt.Sprintf("Repo should be %v, but it is %v", expectedRepo, repo))
	}

	// Test external repository with no tag.
	// docker/distribution/reference will prefix hostname when absent
	image = "kabanero/kabanero-image"
	repo, err = GetImageRepository(image)
	if err != nil {
		t.Fatal(fmt.Sprintf("Unexpected error while removing tag from image: %v. Error: ", err))
	}
	expectedRepo = "docker.io/kabanero/kabanero-image"
	if repo != expectedRepo {
		t.Fatal(fmt.Sprintf("Repo should be %v, but it is %v", expectedRepo, repo))
	}

	// Test internal respository (with port) and tag
	image = "image-registry.openshift-image-registry.svc:5000/kabanero/java-microprofile:1.2.3"
	repo, err = GetImageRepository(image)
	if err != nil {
		t.Fatal(fmt.Sprintf("Unexpected error while removing tag from image: %v. Error: ", err))
	}

	expectedRepo = "image-registry.openshift-image-registry.svc:5000/kabanero/java-microprofile"
	if repo != expectedRepo {
		t.Fatal(fmt.Sprintf("Repo should be %v, but it is %v", expectedRepo, repo))
	}

	// Test internal respository (with port) and no tag
	image = "image-registry.openshift-image-registry.svc:5000/kabanero/java-microprofile"
	repo, err = GetImageRepository(image)
	if err != nil {
		t.Fatal(fmt.Sprintf("Unexpected error while removing tag from image: %v. Error: ", err))
	}

	expectedRepo = "image-registry.openshift-image-registry.svc:5000/kabanero/java-microprofile"
	if repo != expectedRepo {
		t.Fatal(fmt.Sprintf("Repo should be %v, but it is %v", expectedRepo, repo))
	}

	// Test default (?) repository and tag
	image = "java-microprofile:1.2.3"
	repo, err = GetImageRepository(image)
	if err != nil {
		t.Fatal(fmt.Sprintf("Unexpected error while removing tag from image: %v. Error: ", err))
	}

	// docker/distribution/reference will prefix hostname/library when hostname/org absent
	expectedRepo = "docker.io/library/java-microprofile"
	if repo != expectedRepo {
		t.Fatal(fmt.Sprintf("Repo should be %v, but it is %v", expectedRepo, repo))
	}

	// Test default (?) repository and no tag
	image = "java-microprofile"
	repo, err = GetImageRepository(image)
	if err != nil {
		t.Fatal(fmt.Sprintf("Unexpected error while removing tag from image: %v. Error: ", err))
	}

	// docker/distribution/reference will prefix hostname/library when hostname/org absent
	expectedRepo = "docker.io/library/java-microprofile"
	if repo != expectedRepo {
		t.Fatal(fmt.Sprintf("Repo should be %v, but it is %v", expectedRepo, repo))
	}
}

// Tests that GetImageRegistry returns teh registry/domain portion of the input image.
func TestGetImageRegistry(t *testing.T) {
	// Test 1.
	image := "image-registry.openshift-image-registry.svc:5000/kabanero/java-microprofile:1.2.3"
	registry, err := GetImageRegistry(image)
	expectedReg := "image-registry.openshift-image-registry.svc:5000"
	if err != nil {
		t.Fatal(fmt.Sprintf("A registry was expected. An error was received instead. Image: %v. Expected registry: %v. Error: %v", image, expectedReg, err))
	}
	if registry != expectedReg {
		t.Fatal(fmt.Sprintf("The registry retrieved was %v, but it was expected to be: %v", registry, expectedReg))
	}

	// Test 2.
	image = "image-registry-openshift-image-registry-svc:5000/kabanero/java-microprofile:1.2.3"
	registry, err = GetImageRegistry(image)
	expectedReg = "image-registry-openshift-image-registry-svc:5000"
	if err != nil {
		t.Fatal(fmt.Sprintf("A registry was expected. An error was received instead. Image: %v. Expected registry: %v. Error: %v", image, expectedReg, err))
	}
	if registry != expectedReg {
		t.Fatal(fmt.Sprintf("The registry retrieved was %v, but it was expected to be: %v", registry, expectedReg))
	}

	// Test 3.
	image = "image-registry_openshift-image-registry__svc:5000/kabanero/java-microprofile:123"
	registry, err = GetImageRegistry(image)
	if err == nil {
		t.Fatal(fmt.Sprintf("A error was expected. A registry was received instead. Image: %v. Registry: %v.", image, registry))
	}

	// Test 4.
	image = "image-registry.openshift-image-registry_svc:5000/kabanero/java-microprofile:123"
	registry, err = GetImageRegistry(image)
	if err == nil {
		t.Fatal(fmt.Sprintf("A error was expected. A registry was received instead. Image: %v. Registry: %v.", image, registry))
	}

	// Test 5.
	image = "image-registry-openshift-image-registry-svc:5000/kabanero/java-microprofile:1.2.3"
	registry, err = GetImageRegistry(image)
	expectedReg = "image-registry-openshift-image-registry-svc:5000"
	if err != nil {
		t.Fatal(fmt.Sprintf("A registry was expected. An error was received instead. Image: %v. Expected registry: %v. Error: %v", image, expectedReg, err))
	}
	if registry != expectedReg {
		t.Fatal(fmt.Sprintf("The registry retrieved was %v, but it was expected to be: %v", registry, expectedReg))
	}

	// Test 6.
	image = "docker.io/kabanero/kabanero-image:1.2.3"
	registry, err = GetImageRegistry(image)
	expectedReg = "docker.io"
	if err != nil {
		t.Fatal(fmt.Sprintf("A registry was expected. An error was received instead. Image: %v. Expected registry: %v. Error: %v", image, expectedReg, err))
	}
	if registry != expectedReg {
		t.Fatal(fmt.Sprintf("The registry retrieved was %v, but it was expected to be: %v", registry, expectedReg))
	}

	// Test 7.
	image = "my-registry.io/kabanero/kabanero-image:1.2.3"
	registry, err = GetImageRegistry(image)
	expectedReg = "my-registry.io"
	if err != nil {
		t.Fatal(fmt.Sprintf("A registry was expected. An error was received instead. Image: %v. Expected registry: %v. Error: %v", image, expectedReg, err))
	}
	if registry != expectedReg {
		t.Fatal(fmt.Sprintf("The registry retrieved was %v, but it was expected to be: %v", registry, expectedReg))
	}

	// Test 8. Default registry expected.
	image = "some_path-component/kabanero/kabanero-image:1.2.3"
	registry, err = GetImageRegistry(image)
	expectedReg = "docker.io"
	if err != nil {
		t.Fatal(fmt.Sprintf("A registry was expected. An error was received instead. Image: %v. Expected registry: %v. Error: %v", image, expectedReg, err))
	}
	if registry != expectedReg {
		t.Fatal(fmt.Sprintf("The registry retrieved was %v, but it was expected to be: %v", registry, expectedReg))
	}

	// Test 9.
	image = "my-registry.io/kabanero/kabanero-image:1.2.3"
	registry, err = GetImageRegistry(image)
	expectedReg = "my-registry.io"
	if err != nil {
		t.Fatal(fmt.Sprintf("A registry was expected. An error was received instead. Image: %v. Expected registry: %v. Error: %v", image, expectedReg, err))
	}
	if registry != expectedReg {
		t.Fatal(fmt.Sprintf("The registry retrieved was %v, but it was expected to be: %v", registry, expectedReg))
	}

	// Test 10.
	image = "my-registry--is--great.io/kabanero/kabanero-image:1.2.3"
	registry, err = GetImageRegistry(image)
	expectedReg = "my-registry--is--great.io"
	if err != nil {
		t.Fatal(fmt.Sprintf("A registry was expected. An error was received instead. Image: %v. Expected registry: %v. Error: %v", image, expectedReg, err))
	}
	if registry != expectedReg {
		t.Fatal(fmt.Sprintf("The registry retrieved was %v, but it was expected to be: %v", registry, expectedReg))
	}

	// Test 11.
	image = "my-registry-is_great.io/kabanero/kabanero-image:1.2.3"
	registry, err = GetImageRegistry(image)
	expectedReg = ""
	if err != nil {
		t.Fatal(fmt.Sprintf("A registry was expected. An error was received instead. Image: %v. Expected registry: %v. Error: %v", image, expectedReg, err))
	}
	// Invalid hostnames end up parsed as the org, nor does it fill in docker.io as the domain
	// https://github.com/docker/distribution/blob/master/reference/reference_test.go#L164
	if registry != expectedReg {
		t.Fatal(fmt.Sprintf("The registry retrieved was %v, but it was expected to be: %v", registry, expectedReg))
	}

	// Test 12.
	image = "kabanero/kabanero/kabanero-image:1.2.3"
	registry, err = GetImageRegistry(image)
	expectedReg = "docker.io"
	if err != nil {
		t.Fatal(fmt.Sprintf("A registry was expected. An error was received instead. Image: %v. Expected registry: %v. Error: %v", image, expectedReg, err))
	}
	if registry != expectedReg {
		t.Fatal(fmt.Sprintf("The registry retrieved was %v, but it was expected to be: %v", registry, expectedReg))
	}
}
