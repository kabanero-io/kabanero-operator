package stack

import (
	"testing"

	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Base stack
var mutatingBaseStack kabanerov1alpha2.Stack = kabanerov1alpha2.Stack{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "java-microprofile",
		Namespace: "Kabanero",
		UID:       "1",
		OwnerReferences: []metav1.OwnerReference{
			metav1.OwnerReference{
				APIVersion: "a/1",
				Kind:       "Kabanero",
				Name:       "kabanero",
				UID:        "1",
			},
		},
	},
	Spec: kabanerov1alpha2.StackSpec{
		Name: "java-microprofile",
		Versions: []kabanerov1alpha2.StackVersion{{
			Pipelines:    []kabanerov1alpha2.PipelineSpec{},
			Version:      "1.2.3",
			DesiredState: "active",
			Images: []kabanerov1alpha2.Image{{
				Id:    "java-microprofile",
				Image: "kabanero/java-microprofile:1.2"}}}},
	},
}

// Current stack.Spec = New stack.Spec.
// Current stack.Spec.Versions[] (empty) =  and New stack.Spec.Versions[].
// Expectation: stack.Spec.versions[0] should be added with the contents of stack.Spec data.
func Test1(t *testing.T) {
	newStack := mutatingBaseStack.DeepCopy()
	err := processUpdate(&mutatingBaseStack, newStack)
	if err != nil {
		t.Fatal("Unexpected error during mutation.", err)
	}

	expectedversion0 := kabanerov1alpha2.StackVersion{
		Pipelines:    []kabanerov1alpha2.PipelineSpec{},
		Version:      "1.2.3",
		DesiredState: "active",
		Images: []kabanerov1alpha2.Image{{
			Id:    "java-microprofile",
			Image: "kabanero/java-microprofile"}},
	}

	if newStack.Spec.Versions[0].Version != expectedversion0.Version {
		t.Fatal("Mutated versions[0] does not match expected versions[0] values. Mutated versions[0]: ", newStack.Spec.Versions[0], "Expected versions[0]: ", expectedversion0)
	}

	if newStack.Spec.Versions[0].DesiredState != expectedversion0.DesiredState {
		t.Fatal("Mutated versions[0] does not match expected versions[0] values. Mutated versions[0]: ", newStack.Spec.Versions[0], "Expected versions[0]: ", expectedversion0)
	}

	if newStack.Spec.Versions[0].Images[0].Image != expectedversion0.Images[0].Image {
		t.Fatal("Mutated versions[0].Images[0].Image does not match expected versions[0].Images[0].Image  values. Mutated versions[0].Images[0].Image: ", newStack.Spec.Versions[0].Images[0].Image, "Expected versions[0].Images[0].Image: ", expectedversion0.Images[0].Image)
	}
}
