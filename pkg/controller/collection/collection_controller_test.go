package collection

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var cPipelines []kabanerov1alpha1.PipelineStatus = []kabanerov1alpha1.PipelineStatus{
	{Name: "commonPipeline", Url: "http://pipelinelink", Digest: "abc121cba"}}

var cImages []kabanerov1alpha1.Image = []kabanerov1alpha1.Image{
	{Id: "a1b22b1a", Image: "some/pipeline:1.2.3"}}

var cInstance *kabanerov1alpha1.Collection = &kabanerov1alpha1.Collection{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "java-microprofile",
		Namespace: "Kabanero",
		OwnerReferences: []metav1.OwnerReference{
			metav1.OwnerReference{
				APIVersion: "a/1",
				Kind:       "Kabanero",
				Name:       "kabanero",
				UID:        "1234567890",
						},
					},
						},
	Spec: kabanerov1alpha1.CollectionSpec{
		Name:                 "java-microprofile",
		Version:              "1.2.3",
		RepositoryUrl:        "https://java-microprofile",
		SkipCertVerification: true,
		DesiredState:         "active",
		Versions: []kabanerov1alpha1.CollectionVersion{
			{
				Version:              "1.2.3",
				RepositoryUrl:        "https://java-microprofile/1.2.3",
				SkipCertVerification: true,
				DesiredState:         "active",
					},
			{
				Version:              "4.5.6",
				RepositoryUrl:        "https://java-microprofile/4.5.6",
				SkipCertVerification: true,
				DesiredState:         "active",
			}},
					},
	Status: kabanerov1alpha1.CollectionStatus{
		Status:          "active",
		ActivePipelines: cPipelines,
		Images:          cImages,
		Versions: []kabanerov1alpha1.CollectionVersionStatus{
			{
				Version:   "1.2.3",
				Pipelines: cPipelines,
				Images:    cImages,
				},
			{
				Version:   "4.5.6",
				Pipelines: cPipelines,
				Images:    cImages,
			},
		},
	},
}

var sInstance *kabanerov1alpha2.Stack = &kabanerov1alpha2.Stack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "java-microprofile",
			Namespace: "Kabanero",
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: "a/1",
					Kind:       "Kabanero",
					Name:       "kabanero",
				UID:        "1234567890",
				},
			},
		},
	Spec: kabanerov1alpha2.StackSpec{
			Name:     "java-microprofile",
		Versions: []kabanerov1alpha2.StackVersion{
			{
		Version:   "1.2.3",
				SkipCertVerification: true,
				DesiredState:         "active",
				Pipelines:            []kabanerov1alpha2.PipelineSpec{{Id: "commonPipeline", Sha256: "abc121cba", Url: "http://pipelinelink"}},
				Images:               []kabanerov1alpha2.Image{{Id: "a1b22b1a", Image: "some/pipeline:1.2.3"}},
		},
			{
				Version:              "4.5.6",
				SkipCertVerification: true,
				DesiredState:         "active",
				Pipelines:            []kabanerov1alpha2.PipelineSpec{{Id: "commonPipeline", Sha256: "abc121cba", Url: "http://pipelinelink"}},
				Images:               []kabanerov1alpha2.Image{{Id: "a1b22b1a", Image: "some/pipeline:1.2.3"}},
		},
		},
		},
	}

var kInstance *kabanerov1alpha1.Kabanero = &kabanerov1alpha1.Kabanero{
	TypeMeta: metav1.TypeMeta{Kind: "Kabanero",
		APIVersion: "a/1"},
	ObjectMeta: metav1.ObjectMeta{Name: "kabanero", UID: "1234567890"},
	Spec: kabanerov1alpha1.KabaneroSpec{
		Version: "4.5.6",
		Collections: kabanerov1alpha1.InstanceCollectionConfig{
			Repositories: []kabanerov1alpha1.RepositoryConfig{{
				Name:                       "incubator",
				Url:                        "https://github.com/kabanero-io/collections/releases/download/0.4.0/kabanero-index.yaml",
				ActivateDefaultCollections: true,
				SkipCertVerification:       false,
			}},
		},
		},
	}

var r *ReconcileCollection = &ReconcileCollection{indexResolver: ResolveIndex}

// Tests a fully deployed collection (status filled in) conversion to stack structure.
func TestConvertNormalCollectionToStack(t *testing.T) {

	expectedStack := *sInstance
	stack, err := r.convertCollectionToStack(kInstance, cInstance)

	if err != nil {
		t.Fatal(err)
	}

	if stack.ObjectMeta.Name != expectedStack.ObjectMeta.Name {
		t.Fatal(fmt.Sprintf("The expected stack.ObjectMeta.Name does not match resulting stack.ObjectMeta.Name. Converted: %v. Expected: %v", stack.ObjectMeta.Name, expectedStack.ObjectMeta.Name))
	}
	if stack.ObjectMeta.Namespace != expectedStack.ObjectMeta.Namespace {
		t.Fatal(fmt.Sprintf("The expected stack.ObjectMeta.Namespace does not match resulting stack.ObjectMeta.Namespace. Converted: %v. Expected: %v", stack.ObjectMeta.Namespace, expectedStack.ObjectMeta.Namespace))
	}
	if len(stack.ObjectMeta.OwnerReferences) != 1 {
		t.Fatal(fmt.Sprintf("The resulting stack.ObjectMeta.OwnerReferences does not contain an owner entry. It should have one."))
	}
	if stack.ObjectMeta.OwnerReferences[0].Kind != kInstance.TypeMeta.Kind {
		t.Fatal(fmt.Sprintf("The resulting stack.ObjectMeta.OwnerReferences[0].Kind does not match expected kInstance.TypeMeta.Kind. Converted %v. Expected %v", stack.ObjectMeta.OwnerReferences[0].Kind, kInstance.TypeMeta.Kind))
	}
	if stack.ObjectMeta.OwnerReferences[0].UID != kInstance.ObjectMeta.UID {
		t.Fatal(fmt.Sprintf("The resulting stack.ObjectMeta.OwnerReferences[0].UID does not match expected kInstance.ObjectMeta.UID. Converted %v. Expected %v", stack.ObjectMeta.OwnerReferences[0].UID, kInstance.ObjectMeta.UID))
	}
	if !cmp.Equal(stack.Spec, expectedStack.Spec) {
		t.Fatal(fmt.Sprintf("The expected stack and resulting stack are not the same. Coverted: %v. Expected: %v", stack, expectedStack))
	}
	}

// Tests the conversion of a collection resource with no versions[].
func TestConvertCollectionNoVersionsToStack(t *testing.T) {
	expectedStack := sInstance.DeepCopy()
	expectedStack.Spec.Versions = []kabanerov1alpha2.StackVersion{
		{
			Version:              "1.2.3",
			SkipCertVerification: true,
			DesiredState:         "active",
			Pipelines:            []kabanerov1alpha2.PipelineSpec{{Id: "commonPipeline", Sha256: "abc121cba", Url: "http://pipelinelink"}},
			Images:               []kabanerov1alpha2.Image{{Id: "a1b22b1a", Image: "some/pipeline:1.2.3"}},
	}}
	tc := cInstance.DeepCopy()
	tc.Spec.Versions = nil

	stack, err := r.convertCollectionToStack(kInstance, tc)
	if err != nil {
		t.Fatal(err)
	}

	if stack.ObjectMeta.Name != expectedStack.ObjectMeta.Name {
		t.Fatal(fmt.Sprintf("The expected stack.ObjectMeta.Name does not match resulting stack.ObjectMeta.Name. Converted: %v. Expected: %v", stack.ObjectMeta.Name, expectedStack.ObjectMeta.Name))
	}
	if stack.ObjectMeta.Namespace != expectedStack.ObjectMeta.Namespace {
		t.Fatal(fmt.Sprintf("The expected stack.ObjectMeta.Namespace does not match resulting stack.ObjectMeta.Namespace. Converted: %v. Expected: %v", stack.ObjectMeta.Namespace, expectedStack.ObjectMeta.Namespace))
		}
	if len(stack.ObjectMeta.OwnerReferences) != 1 {
		t.Fatal(fmt.Sprintf("The resulting stack.ObjectMeta.OwnerReferences does not contain an owner entry. It should have one."))
		}
	if stack.ObjectMeta.OwnerReferences[0].Kind != kInstance.TypeMeta.Kind {
		t.Fatal(fmt.Sprintf("The resulting stack.ObjectMeta.OwnerReferences[0].Kind does not match expected kInstance.TypeMeta.Kind. Converted %v. Expected %v", stack.ObjectMeta.OwnerReferences[0].Kind, kInstance.TypeMeta.Kind))
			}
	if stack.ObjectMeta.OwnerReferences[0].UID != kInstance.ObjectMeta.UID {
		t.Fatal(fmt.Sprintf("The resulting stack.ObjectMeta.OwnerReferences[0].UID does not match expected kInstance.ObjectMeta.UID. Converted %v. Expected %v", stack.ObjectMeta.OwnerReferences[0].UID, kInstance.ObjectMeta.UID))
			}
	if !cmp.Equal(stack.Spec, expectedStack.Spec) {
		t.Fatal(fmt.Sprintf("The expected stack and resulting stack are not the same. Coverted: %v. Expected: %v", stack, expectedStack))
		}
	}

// Tests the conversion of a collection resource where both versions do not have pipelines and images in the status section.
// The pipeline and image data are obtained from the kabnanero instance's index URL.
func TestConvertToStackUsingCollectionWithEmptyPipelineAndImageAllVersions(t *testing.T) {
	expectedStack := *sInstance
	tc := cInstance.DeepCopy()
	tc.Status.ActivePipelines = nil
	tc.Status.Images = nil
	tc.Spec.Version = "0.2.19"
	tc.Status.Versions[0].Pipelines = nil
	tc.Status.Versions[0].Images = nil
	tc.Spec.Versions[0].Version = "0.2.19"
	tc.Status.Versions[1].Pipelines = nil
	tc.Status.Versions[1].Images = nil
	tc.Spec.Versions[1].Version = "0.2.19"

	stack, err := r.convertCollectionToStack(kInstance, tc)
	if err != nil {
		t.Fatal(err)
	}

	if stack.ObjectMeta.Name != expectedStack.ObjectMeta.Name {
		t.Fatal(fmt.Sprintf("The expected stack.ObjectMeta.Name does not match resulting stack.ObjectMeta.Name. Converted: %v. Expected: %v", stack.ObjectMeta.Name, expectedStack.ObjectMeta.Name))
	}
	if stack.ObjectMeta.Namespace != expectedStack.ObjectMeta.Namespace {
		t.Fatal(fmt.Sprintf("The expected stack.ObjectMeta.Namespace does not match resulting stack.ObjectMeta.Namespace. Converted: %v. Expected: %v", stack.ObjectMeta.Namespace, expectedStack.ObjectMeta.Namespace))
		}
	if len(stack.ObjectMeta.OwnerReferences) != 1 {
		t.Fatal(fmt.Sprintf("The resulting stack.ObjectMeta.OwnerReferences does not contain an owner entry. It should have one."))
		}
	if stack.ObjectMeta.OwnerReferences[0].Kind != kInstance.TypeMeta.Kind {
		t.Fatal(fmt.Sprintf("The resulting stack.ObjectMeta.OwnerReferences[0].Kind does not match expected kInstance.TypeMeta.Kind. Converted %v. Expected %v", stack.ObjectMeta.OwnerReferences[0].Kind, kInstance.TypeMeta.Kind))
			}
	if stack.ObjectMeta.OwnerReferences[0].UID != kInstance.ObjectMeta.UID {
		t.Fatal(fmt.Sprintf("The resulting stack.ObjectMeta.OwnerReferences[0].UID does not match expected kInstance.ObjectMeta.UID. Converted %v. Expected %v", stack.ObjectMeta.OwnerReferences[0].UID, kInstance.ObjectMeta.UID))
	}

	// This version of the collection had no status.
	if len(stack.Spec.Versions[0].Images) == 0 {
		t.Fatal(fmt.Sprintf("The resulting stack.Spec.Versions[0].Images array is empty. An entry was expected."))
	}
	if stack.Spec.Versions[0].Images[0].Id != "java-microprofile" {
		t.Fatal(fmt.Sprintf("The resulting stack.Spec.Versions[0].Images[0].Id is not java-microprofile as expected. It instead shows: " + stack.Spec.Versions[0].Images[0].Id))
	}

	if len(stack.Spec.Versions[0].Pipelines) == 0 {
		t.Fatal(fmt.Sprintf("The resulting stack.Spec.Versions[0].Pipelines array is empty. An entry was expected."))
	}
	if stack.Spec.Versions[0].Pipelines[0].Id != "default" {
		t.Fatal(fmt.Sprintf("The resulting stack.Spec.Versions[0].Pipelines[0].Id is not default as expected. It instead shows: " + stack.Spec.Versions[0].Pipelines[0].Id))
}

	// This version of the collection had a status.
	if len(stack.Spec.Versions[1].Images) == 0 {
		t.Fatal(fmt.Sprintf("The resulting stack.Spec.Versions[1].Images array is empty. An entry was expected."))
	}
	if stack.Spec.Versions[1].Images[0].Id != "java-microprofile" {
		t.Fatal(fmt.Sprintf("The resulting stack.Spec.Versions[1].Images[0].Id is not java-microprofile as expected. It instead shows: " + stack.Spec.Versions[0].Images[0].Id))
	}

	if len(stack.Spec.Versions[1].Pipelines) == 0 {
		t.Fatal(fmt.Sprintf("The resulting stack.Spec.Versions[1].Pipelines array is empty. An entry was expected."))
	}
	if stack.Spec.Versions[1].Pipelines[0].Id != "default" {
		t.Fatal(fmt.Sprintf("The resulting stack.Spec.Versions[1].Pipelines[0].Id is not default as expected. It instead shows: " + stack.Spec.Versions[0].Pipelines[0].Id))
		}
		}

// Tests the conversion of a collection resource where one version does not have pipelines and images in the status section.
// The pipeline and image data are obtained from the kabnanero instance's index URL.
// The second version has pipeline and image entries defined in the status section. The data is taken from the status.
func TestConvertToStackUsingCollectionWithOneVersionHasEmptyPipelineAndImage(t *testing.T) {
	expectedStack := *sInstance
	tc := cInstance.DeepCopy()
	tc.Status.ActivePipelines = nil
	tc.Status.Images = nil
	tc.Spec.Version = "0.2.19"
	tc.Status.Versions[0].Pipelines = nil
	tc.Status.Versions[0].Images = nil
	tc.Spec.Versions[0].Version = "0.2.19"

	stack, err := r.convertCollectionToStack(kInstance, tc)
	if err != nil {
		t.Fatal(err)
	}

	if stack.ObjectMeta.Name != expectedStack.ObjectMeta.Name {
		t.Fatal(fmt.Sprintf("The expected stack.ObjectMeta.Name does not match resulting stack.ObjectMeta.Name. Converted: %v. Expected: %v", stack.ObjectMeta.Name, expectedStack.ObjectMeta.Name))
	}
	if stack.ObjectMeta.Namespace != expectedStack.ObjectMeta.Namespace {
		t.Fatal(fmt.Sprintf("The expected stack.ObjectMeta.Namespace does not match resulting stack.ObjectMeta.Namespace. Converted: %v. Expected: %v", stack.ObjectMeta.Namespace, expectedStack.ObjectMeta.Namespace))
		}
	if len(stack.ObjectMeta.OwnerReferences) != 1 {
		t.Fatal(fmt.Sprintf("The resulting stack.ObjectMeta.OwnerReferences does not contain an owner entry. It should have one."))
		}
	if stack.ObjectMeta.OwnerReferences[0].Kind != kInstance.TypeMeta.Kind {
		t.Fatal(fmt.Sprintf("The resulting stack.ObjectMeta.OwnerReferences[0].Kind does not match expected kInstance.TypeMeta.Kind. Converted %v. Expected %v", stack.ObjectMeta.OwnerReferences[0].Kind, kInstance.TypeMeta.Kind))
		}
	if stack.ObjectMeta.OwnerReferences[0].UID != kInstance.ObjectMeta.UID {
		t.Fatal(fmt.Sprintf("The resulting stack.ObjectMeta.OwnerReferences[0].UID does not match expected kInstance.ObjectMeta.UID. Converted %v. Expected %v", stack.ObjectMeta.OwnerReferences[0].UID, kInstance.ObjectMeta.UID))
	}

	// This version of the collection had no status.
	if len(stack.Spec.Versions[0].Images) == 0 {
		t.Fatal(fmt.Sprintf("The resulting stack.Spec.Versions[0].Images array is empty. An entry was expected."))
	}
	if stack.Spec.Versions[0].Images[0].Id != "java-microprofile" {
		t.Fatal(fmt.Sprintf("The resulting stack.Spec.Versions[0].Images[0].Id is not java-microprofile as expected. It instead shows: " + stack.Spec.Versions[0].Images[0].Id))
}

	if len(stack.Spec.Versions[0].Pipelines) == 0 {
		t.Fatal(fmt.Sprintf("The resulting stack.Spec.Versions[0].Pipelines array is empty. An entry was expected."))
	}
	if stack.Spec.Versions[0].Pipelines[0].Id != "default" {
		t.Fatal(fmt.Sprintf("The resulting stack.Spec.Versions[0].Pipelines[0].Id is not default as expected. It instead shows: " + stack.Spec.Versions[0].Pipelines[0].Id))
	}

	// This version of the collection had a status.
	if len(stack.Spec.Versions[1].Images) == 0 {
		t.Fatal(fmt.Sprintf("The resulting stack.Spec.Versions[1].Images array is empty. An entry was expected."))
	}
	if stack.Spec.Versions[1].Images[0].Id != "a1b22b1a" {
		t.Fatal(fmt.Sprintf("The resulting stack.Spec.Versions[1].Images[0].Id is not a1b22b1a as expected. It instead shows: " + stack.Spec.Versions[0].Images[0].Id))
			}

	if len(stack.Spec.Versions[1].Pipelines) == 0 {
		t.Fatal(fmt.Sprintf("The resulting stack.Spec.Versions[1].Pipelines array is empty. An entry was expected."))
			}
	if stack.Spec.Versions[1].Pipelines[0].Id != "commonPipeline" {
		t.Fatal(fmt.Sprintf("The resulting stack.Spec.Versions[1].Pipelines[0].Id is not commonPipeline as expected. It instead shows: " + stack.Spec.Versions[0].Pipelines[0].Id))
			}
			}

// Tests the conversion of a collection with no Images and Pipelines defined and the index.yaml does not
// contain the version of the collection. The collection to stack conversion should fail.
func TestConvertToStackUsingCollectionWithEmptyPipelineAndImageVersionNotFound(t *testing.T) {
	tc := cInstance.DeepCopy()
	tc.Status.ActivePipelines = nil
	tc.Status.Images = nil
	tc.Status.Versions[0].Pipelines = nil
	tc.Status.Versions[0].Images = nil

	stack, err := r.convertCollectionToStack(kInstance, tc)
	if err == nil {
		t.Fatal("An error was expected to be thrown because the index did not contain a stack with the name specified in the collection.")
				}
	if stack != nil {
		t.Fatal("A nil stack was expected because there was an error. Instead, the following was found stack was found: ", stack)
		}
	}

// Tests the conversion of a collection with inactive status and a name that could not be found in the index.yaml
// associated with the kabanero instance.
func TestConvertToStackUsingCollectionWithInactiveStateAndUnknownStack(t *testing.T) {
	tc := cInstance.DeepCopy()
	tc.Status.Status = "inactive"
	tc.ObjectMeta.Name = "someBogusName"
	tc.Spec.Name = "someBogusName"
	stack, err := r.convertCollectionToStack(kInstance, tc)
	if err == nil {
		t.Fatal("An error was expected to be thrown because the index did not contain a stack with the name specified in the collection.")
	}
	if stack != nil {
		t.Fatal("A nil stack was expected because there was an error. Instead, the following was found stack was found: ", stack)
		}
		}
