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
			DesiredState: "active",
			Version:      "1.2.3",
			Pipelines: []kabanerov1alpha2.PipelineSpec{{
				Sha256: "abc121cba",
				Https: kabanerov1alpha2.HttpsProtocolFile{
					Url: "http://pipelinelink/pipeline.tar.gz",
				},
			}},
			Images: []kabanerov1alpha2.Image{{
				Id:    "java-microprofile",
				Image: "kabanero/java-microprofile",
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
		t.Fatal("Validation should have passed and the stack update should have been allowed. Error: ", err)
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
		t.Fatal("Validation should have failed. The stack update/create was incorrectly allowed.")
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
		t.Fatal("Validation should have failed. The stack update/create was incorrectly allowed.")
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
		t.Fatal("Validation should have passed and the stack update should have been allowed. Error: ", err)
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
		t.Fatal("Validation should have passed and the stack update should have been allowed. Error: ", err)
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
		t.Fatal("Validation should have failed. The stack update/create was incorrectly allowed.")
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
		t.Fatal("Validation should have failed. The stack update/create was incorrectly allowed.")
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
		t.Fatal("Validation should have failed. The stack update/create was incorrectly allowed.")
	}

	if len(msg) == 0 {
		t.Fatal("Validation failed. A message was expected: ", msg)
	}

	if err == nil {
		t.Fatal("Validation failed. An error was expected: ", err)
	}
}

// Spec.Versions[].Pipelines[].Sha256 empty with tar.gz
func TestValidatingWebhook9(t *testing.T) {
	newStack := validatingStack.DeepCopy()
	newStack.Spec.Versions[0].Pipelines[0].Sha256 = ""

	cv := stackValidator{}
	allowed, msg, err := cv.validateStackFn(nil, newStack)

	if allowed {
		t.Fatal("Validation should have failed. The stack update/create was incorrectly allowed.")
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
		t.Fatal("Validation should have failed. The stack update/create was incorrectly allowed.")
	}

	if len(msg) == 0 {
		t.Fatal("Validation failed. A message was expected: ", msg)
	}

	if err == nil {
		t.Fatal("Validation failed. An error was expected: ", err)
	}
}

// Spec.Versions[].Images[] is empty
func TestValidatingWebhook11(t *testing.T) {
	newStack := validatingStack.DeepCopy()
	newStack.Spec.Versions[0].Images = []kabanerov1alpha2.Image{}

	cv := stackValidator{}
	allowed, msg, err := cv.validateStackFn(nil, newStack)

	if allowed {
		t.Fatal("Validation should have failed because Spec.Versions[].Images[] is empty. The stack update/create was incorrectly allowed.")
	}

	if len(msg) == 0 {
		t.Fatal("Validation failed. A message was expected: ", msg)
	}

	if err == nil {
		t.Fatal("Validation failed. An error was expected: ", err)
	}
}

// Spec.Versions[].Images[].Image has a tag
func TestValidatingWebhook12(t *testing.T) {
	newStack := validatingStack.DeepCopy()
	newStack.Spec.Versions[0].Images[0].Image = "kabanero/java-microprofile:1.2.3"

	cv := stackValidator{}
	allowed, msg, err := cv.validateStackFn(nil, newStack)

	if allowed {
		t.Fatal("Validation should have failed because the stack's image has a tag. The stack update/create was incorrectly allowed.")
	}

	if len(msg) == 0 {
		t.Fatal("Validation failed. A message was expected: ", msg)
	}

	if err == nil {
		t.Fatal("Validation failed. An error was expected: ", err)
	}
}

// Spec.Versions[].Images[].Image contains a port, not a tag
func TestValidatingWebhook13(t *testing.T) {
	newStack := validatingStack.DeepCopy()
	newStack.Spec.Versions[0].Images[0].Image = "image-registry.openshift-image-registry.svc:5000/kabanero/java-microprofile"

	cv := stackValidator{}
	allowed, msg, err := cv.validateStackFn(nil, newStack)

	if !allowed {
		t.Fatal("Validation should not have failed because the stack's image has no tag. The stack update/create was incorrectly denied.")
	}

	if len(msg) != 0 {
		t.Fatal("Validation passed. A message was not expected: ", msg)
	}

	if err != nil {
		t.Fatal("Validation passed. An error was not expected: ", err)
	}
}

// Spec.Versions[].Images[].Image has a port and tag
func TestValidatingWebhook14(t *testing.T) {
	newStack := validatingStack.DeepCopy()
	newStack.Spec.Versions[0].Images[0].Image = "image-registry.openshift-image-registry.svc:5000/kabanero/java-microprofile:latest"

	cv := stackValidator{}
	allowed, msg, err := cv.validateStackFn(nil, newStack)

	if allowed {
		t.Fatal("Validation should have failed because the stack's image has a tag. The stack update/create was incorrectly allowed.")
	}

	if len(msg) == 0 {
		t.Fatal("Validation failed. A message was expected: ", msg)
	}

	if err == nil {
		t.Fatal("Validation failed. An error was expected: ", err)
	}
}


// Yaml file
func TestValidatingWebhook15(t *testing.T) {
	newStack := validatingStack.DeepCopy()
	newStack.Spec.Versions[0].Pipelines[0].Https.Url = "http://pipelinelink/pipeline.yaml"

	cv := stackValidator{}
	allowed, msg, err := cv.validateStackFn(nil, newStack)

	if !allowed {
		t.Fatal("Validation should have passed and the stack update should have been allowed. Error: ", err)
	}

	if len(msg) != 0 {
		t.Fatal("Validation succeeded. A message was not expected. Message: ", msg)
	}

	if err != nil {
		t.Fatal("Validation succeeded. An error was not expected. Error: ", err)
	}
}


// Unknown file
func TestValidatingWebhook16(t *testing.T) {
	newStack := validatingStack.DeepCopy()
	newStack.Spec.Versions[0].Pipelines[0].Https.Url = "http://pipelinelink/pipeline.nope"

	cv := stackValidator{}
	allowed, msg, err := cv.validateStackFn(nil, newStack)

	if allowed {
		t.Fatal("Validation should have failed because the file type in the pipeline url is unknown.")
	}

	if len(msg) == 0 {
		t.Fatal("Validation failed. A message was expected: ", msg)
	}

	if err == nil {
		t.Fatal("Validation failed. An error was expected: ", err)
	}
}

// Spec.Versions[].Pipelines[].Sha256 empty with yaml
func TestValidatingWebhook17(t *testing.T) {
	newStack := validatingStack.DeepCopy()
	newStack.Spec.Versions[0].Pipelines[0].Sha256 = ""
	newStack.Spec.Versions[0].Pipelines[0].Https.Url = "http://pipelinelink/pipeline.yaml"

	cv := stackValidator{}
	allowed, msg, err := cv.validateStackFn(nil, newStack)

	if !allowed {
		t.Fatal("Validation should have passed and the stack update should have been allowed. Error: ", err)
	}

	if len(msg) != 0 {
		t.Fatal("Validation succeeded. A message was not expected. Message: ", msg)
	}

	if err != nil {
		t.Fatal("Validation succeeded. An error was not expected. Error: ", err)
	}
}


// Spec.Versions[].Pipelines[].GitRelease .tar.gz
func TestValidatingWebhook18(t *testing.T) {
	newStack := validatingStack.DeepCopy()
	newStack.Spec.Versions[0].Pipelines[0].Https.Url = ""
	newStack.Spec.Versions[0].Pipelines[0].GitRelease = kabanerov1alpha2.GitReleaseSpec{
		Hostname: "somehost",
		Organization: "someorg",
		Project: "someproject",
		Release: "somerelease",
		AssetName: "pipelines.tar.gz",
	}

	cv := stackValidator{}
	allowed, msg, err := cv.validateStackFn(nil, newStack)

	if !allowed {
		t.Fatal("Validation should have passed and the stack update should have been allowed. Error: ", err)
	}

	if len(msg) != 0 {
		t.Fatal("Validation succeeded. A message was not expected. Message: ", msg)
	}

	if err != nil {
		t.Fatal("Validation succeeded. An error was not expected. Error: ", err)
	}
}


// Spec.Versions[].Pipelines[].GitRelease .yaml no sha
func TestValidatingWebhook19(t *testing.T) {
	newStack := validatingStack.DeepCopy()
	newStack.Spec.Versions[0].Pipelines[0].Https.Url = ""
	newStack.Spec.Versions[0].Pipelines[0].Sha256 = ""
	newStack.Spec.Versions[0].Pipelines[0].GitRelease = kabanerov1alpha2.GitReleaseSpec{
		Hostname: "somehost",
		Organization: "someorg",
		Project: "someproject",
		Release: "somerelease",
		AssetName: "pipelines.yaml",
	}

	cv := stackValidator{}
	allowed, msg, err := cv.validateStackFn(nil, newStack)

	if !allowed {
		t.Fatal("Validation should have passed and the stack update should have been allowed. Error: ", err)
	}

	if len(msg) != 0 {
		t.Fatal("Validation succeeded. A message was not expected. Message: ", msg)
	}

	if err != nil {
		t.Fatal("Validation succeeded. An error was not expected. Error: ", err)
	}
}


// Spec.Versions[].Pipelines[].GitRelease unknown file
func TestValidatingWebhook20(t *testing.T) {
	newStack := validatingStack.DeepCopy()
	newStack.Spec.Versions[0].Pipelines[0].Https.Url = ""
	newStack.Spec.Versions[0].Pipelines[0].GitRelease = kabanerov1alpha2.GitReleaseSpec{
		Hostname: "somehost",
		Organization: "someorg",
		Project: "someproject",
		Release: "somerelease",
		AssetName: "pipelines.nope",
	}
	
	cv := stackValidator{}
	allowed, msg, err := cv.validateStackFn(nil, newStack)

	if allowed {
		t.Fatal("Validation should have failed because the file type in the asset file is unknown type.")
	}

	if len(msg) == 0 {
		t.Fatal("Validation failed. A message was expected: ", msg)
	}

	if err == nil {
		t.Fatal("Validation failed. An error was expected: ", err)
	}
}


// Spec.Versions[].Pipelines[].GitRelease tar.gz no sha
func TestValidatingWebhook21(t *testing.T) {
	newStack := validatingStack.DeepCopy()
	newStack.Spec.Versions[0].Pipelines[0].Https.Url = ""
	newStack.Spec.Versions[0].Pipelines[0].Sha256 = ""
	newStack.Spec.Versions[0].Pipelines[0].GitRelease = kabanerov1alpha2.GitReleaseSpec{
		Hostname: "somehost",
		Organization: "someorg",
		Project: "someproject",
		Release: "somerelease",
		AssetName: "pipelines.tar.gz",
	}
	
	cv := stackValidator{}
	allowed, msg, err := cv.validateStackFn(nil, newStack)

	if allowed {
		t.Fatal("Validation should have failed because the sha is missing for .tar.gz.")
	}

	if len(msg) == 0 {
		t.Fatal("Validation failed. A message was expected: ", msg)
	}

	if err == nil {
		t.Fatal("Validation failed. An error was expected: ", err)
	}
}