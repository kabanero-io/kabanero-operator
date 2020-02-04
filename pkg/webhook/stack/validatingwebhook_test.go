package stack

import (
	"testing"

	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Base stack with stack.Spec.Versions[0] defined.
var validatingStack kabanerov1alpha2.Stack = kabanerov1alpha2.Stack{
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
			DesiredState:  "active",
			Version:       "1.2.3",
			Pipelines:     []kabanerov1alpha2.PipelineSpec{{
				Sha256:      "abc121cba",
				Https:       kabanerov1alpha2.HttpsProtocolFile{
					Url:       "http://pipelinelink",
				},
			}},
			Images:        []kabanerov1alpha2.Image{{
				Id:          "java-microprofile",
				Image:       "kabanero/java-microprofile:latest",
			}},
		}},
	},
}

// Fully formed stack
func TestValidatingWebhook1(t *testing.T) {
	newStack := validatingStack.DeepCopy()
	
	cv := stackValidator{}
	allowed, msg, err := cv.validateStackFn(nil, newStack)

	if !allowed {
		t.Fatal("Validation should have passed. The validation was not allowed: ", err)
	}

	if len(msg) != 0 {
		t.Fatal("Validation succeeded. A message was not expected. Message: ", msg)
	}

	if err != nil {
		t.Fatal("Validation succeeded. An error was not expected. Error: ", err)
	}
}

// Spec.Name not set
func TestValidatingWebhook2(t *testing.T) {
	newStack := validatingStack.DeepCopy()
	newStack.Spec.Name = ""

	cv := stackValidator{}
	allowed, msg, err := cv.validateStackFn(nil, newStack)

	if allowed {
		t.Fatal("Validation should have failed. The validation was allowed instead: ", err)
	}

	if len(msg) == 0 {
		t.Fatal("Validation failed. A message was expected: ", msg)
	}

	if err == nil {
		t.Fatal("Validation failed. An error was expected: ", err)
	}
}

// Spec.Versions empty
func TestValidatingWebhook3(t *testing.T) {
	newStack := validatingStack.DeepCopy()
	newStack.Spec.Versions = newStack.Spec.Versions[:len(newStack.Spec.Versions)-1]

	cv := stackValidator{}
	allowed, msg, err := cv.validateStackFn(nil, newStack)

	if allowed {
		t.Fatal("Validation should have failed. The validation was allowed instead: ", err)
	}

	if len(msg) == 0 {
		t.Fatal("Validation failed. A message was expected: ", msg)
	}

	if err == nil {
		t.Fatal("Validation failed. An error was expected: ", err)
	}
}

// Spec.Versions[].DesiredState empty
func TestValidatingWebhook4(t *testing.T) {
	newStack := validatingStack.DeepCopy()
	newStack.Spec.Versions[0].DesiredState = ""

	cv := stackValidator{}
	allowed, msg, err := cv.validateStackFn(nil, newStack)

	if !allowed {
		t.Fatal("Validation should have passed. The validation was not allowed: ", err)
	}

	if len(msg) != 0 {
		t.Fatal("Validation succeeded. A message was not expected. Message: ", msg)
	}

	if err != nil {
		t.Fatal("Validation succeeded. An error was not expected. Error: ", err)
	}
}

// Spec.Versions[].DesiredState active
func TestValidatingWebhook5(t *testing.T) {
	newStack := validatingStack.DeepCopy()
	newStack.Spec.Versions[0].DesiredState = "active"

	cv := stackValidator{}
	allowed, msg, err := cv.validateStackFn(nil, newStack)

	if !allowed {
		t.Fatal("Validation should have passed. The validation was not allowed: ", err)
	}

	if len(msg) != 0 {
		t.Fatal("Validation succeeded. A message was not expected. Message: ", msg)
	}

	if err != nil {
		t.Fatal("Validation succeeded. An error was not expected. Error: ", err)
	}
}


// Spec.Versions[].DesiredState not active/inactive
func TestValidatingWebhook6(t *testing.T) {
	newStack := validatingStack.DeepCopy()
	newStack.Spec.Versions[0].DesiredState = "invalid"

	cv := stackValidator{}
	allowed, msg, err := cv.validateStackFn(nil, newStack)

	if allowed {
		t.Fatal("Validation should have failed. The validation was allowed instead: ", err)
	}

	if len(msg) == 0 {
		t.Fatal("Validation failed. A message was expected:", msg)
	}

	if err == nil {
		t.Fatal("Validation failed. An error was expected: ", err)
	}
}

// Spec.Versions[].Images[] empty
func TestValidatingWebhook7(t *testing.T) {
	newStack := validatingStack.DeepCopy()
	newStack.Spec.Versions[0].Images = newStack.Spec.Versions[0].Images[:len(newStack.Spec.Versions[0].Images)-1]

	cv := stackValidator{}
	allowed, msg, err := cv.validateStackFn(nil, newStack)

	if allowed {
		t.Fatal("Validation should have failed. The validation was allowed instead: ", err)
	}

	if len(msg) == 0 {
		t.Fatal("Validation failed. A message was expected: ", msg)
	}

	if err == nil {
		t.Fatal("Validation failed. An error was expected: ", err)
	}
}

// Spec.Versions[].Pipelines[].Https.Url empty
func TestValidatingWebhook8(t *testing.T) {
	newStack := validatingStack.DeepCopy()
	newStack.Spec.Versions[0].Pipelines[0].Https.Url = ""

	cv := stackValidator{}
	allowed, msg, err := cv.validateStackFn(nil, newStack)

	if allowed {
		t.Fatal("Validation should have failed. The validation was allowed instead: ", err)
	}

	if len(msg) == 0 {
		t.Fatal("Validation failed. A message was expected: ", msg)
	}

	if err == nil {
		t.Fatal("Validation failed. An error was expected: ", err)
	}
}

// Spec.Versions[].Pipelines[].Sha256 empty
func TestValidatingWebhook9(t *testing.T) {
	newStack := validatingStack.DeepCopy()
	newStack.Spec.Versions[0].Pipelines[0].Sha256 = ""

	cv := stackValidator{}
	allowed, msg, err := cv.validateStackFn(nil, newStack)

	if allowed {
		t.Fatal("Validation should have failed. The validation was allowed instead: ", err)
	}

	if len(msg) == 0 {
		t.Fatal("Validation failed. A message was expected: ", msg)
	}

	if err == nil {
		t.Fatal("Validation failed. An error was expected: ", err)
	}
}


// Spec.Versions not semver
func TestValidatingWebhook10(t *testing.T) {
	newStack := validatingStack.DeepCopy()
	newStack.Spec.Versions[0].Version = "1.0"

	cv := stackValidator{}
	allowed, msg, err := cv.validateStackFn(nil, newStack)

	if allowed {
		t.Fatal("Validation should have failed. The validation was allowed instead: ", err)
	}

	if len(msg) == 0 {
		t.Fatal("Validation failed. A message was expected: ", msg)
	}

	if err == nil {
		t.Fatal("Validation failed. An error was expected: ", err)
	}
}