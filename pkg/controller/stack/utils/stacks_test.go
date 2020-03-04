package utils

import (
	"fmt"
	"testing"

	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testStack kabanerov1alpha2.Stack = kabanerov1alpha2.Stack{
	ObjectMeta: metav1.ObjectMeta{UID: "12345", Namespace: "kabanero"},
	Spec: kabanerov1alpha2.StackSpec{
		Name: "java-microprofile",
		Versions: []kabanerov1alpha2.StackVersion{{
			Version:      "0.2.5",
			DesiredState: "active",
			Pipelines: []kabanerov1alpha2.PipelineSpec{{
				Id:     "default",
				Sha256: "1234567890",
				Https:  kabanerov1alpha2.HttpsProtocolFile{Url: "https://test"},
			}},
			Images: []kabanerov1alpha2.Image{{
				Id:    "image1",
				Image: "kabanero/kabanero-image:latest",
			}, {
				Id:    "image2",
				Image: "image-registry.openshift-image-registry.svc:5000/kabanero/java-microprofile:latest",
			}},
		}},
	},
	Status: kabanerov1alpha2.StackStatus{},
}

// Tests that RemoveTagFromStackImages removes the the tag from the images associated to a stack.
func TestRemoveTagFromStackImages(t *testing.T) {
	stack := testStack.DeepCopy()
	RemoveTagFromStackImages(&stack.Spec.Versions[0], "java-microprofile")
	expectedImage := "kabanero/kabanero-image"
	expectedPrivateRepoImage := "image-registry.openshift-image-registry.svc:5000/kabanero/java-microprofile"
	if stack.Spec.Versions[0].Images[0].Image != expectedImage {
		t.Fatal(fmt.Sprintf("Image should be %v, but it is %v", expectedImage, stack.Spec.Versions[0].Images[0].Image))
	}
	if stack.Spec.Versions[0].Images[1].Image != expectedPrivateRepoImage {
		t.Fatal(fmt.Sprintf("Image should be %v, but it is %v", expectedPrivateRepoImage, stack.Spec.Versions[0].Images[1].Image))
	}
}

// Tests that GetImageRepository removes the tag from the input image.  .
func TestGetImageRepository(t *testing.T) {
	image := "kabanero/kabanero-image:1.2.3"
	repo, err := GetImageRepository(image)
	if err != nil {
		t.Fatal(fmt.Sprintf("Unexpected error while removing tag from image: %v. Error: ", err))
	}
	expectedRepo := "kabanero/kabanero-image"
	if repo != expectedRepo {
		t.Fatal(fmt.Sprintf("Repo should be %v, but it is %v", expectedRepo, repo))
	}

	image = "image-registry.openshift-image-registry.svc:5000/kabanero/java-microprofile:1.2.3"
	repo, err = GetImageRepository(image)
	if err != nil {
		t.Fatal(fmt.Sprintf("Unexpected error while removing tag from image: %v. Error: ", err))
	}

	expectedRepo = "image-registry.openshift-image-registry.svc:5000/kabanero/java-microprofile"
	if repo != expectedRepo {
		t.Fatal(fmt.Sprintf("Repo should be %v, but it is %v", expectedRepo, repo))
	}
}
