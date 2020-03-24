package utils

import (
	"fmt"
	"testing"

	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	corev1 "k8s.io/api/core/v1"
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

var testSecret corev1.Secret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "testSecret",
		Namespace: "kabanero",
		Annotations: map[string]string{
			"kabanero.io/git-0": "https://github.mydomain1.com",
			"kabanero.io/git-6": "https://github.mydomain11.com/finance/accounting/",
			"kabanero.io/git-9": "https://github.mydomain111.com/iot"},
	},
	Type: corev1.SecretTypeTLS,
	Data: map[string][]byte{
		"password": []byte{},
	},
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

// Tests that a secret is found when a secret contains a valid annotation key kabanero.io/git-* and the
// annotation value has the desired hostname with no resource path or query string.
func TestSecretFilterHostWithNoExtraPathInAnnotation(t *testing.T) {
	hostname := "github.yourdomain11.com"
	annotationKey := "kabanero.io/git-"
	secret1 := testSecret.DeepCopy()
	secret2 := testSecret.DeepCopy()
	secret2.SetName("testSecret2")
	secret2AnnoMap := map[string]string{
		"kabanero.io/git-0": "https://github.yourdomain1.com",
		"kabanero.io/git-3": "https://github.yourdomain11.com",
		"kabanero.io/git-6": "https://github.yourdomain111.com"}
	secret2.SetAnnotations(secret2AnnoMap)

	secretList := corev1.SecretList{Items: []corev1.Secret{*secret1, *secret2}}
	secret, err := SecretAnnotationFilter(&secretList, hostname, annotationKey)
	if err != nil {
		t.Fatal(fmt.Sprintf("Error retrieving secret associated with hostname: %v and annotation value: %v. Error: %v", hostname, annotationKey, err))
	}
	if secret.Name != secret2.Name {
		t.Fatal(fmt.Sprintf("Secret: %v does not contain an annotation with a value that includes hostname: %v. Expected secret: %v", secret.Name, hostname, secret2.Name))
	}
}

// Tests that a secret is found when a secret contains a valid annotation key kabanero.io/git-* and the
// annotation value has the desired hostname with a resource path and query string.
func TestSecretFilterHostWithExtraPathInAnnotation(t *testing.T) {
	hostname := "github.mydomain11.com"
	annotationKey := "kabanero.io/git-"
	secret1 := testSecret.DeepCopy()
	secret1.Annotations["kabanero.io/git-6"] = "https://github.mydomain11.com/finance/accounting/taxes?id=12345&name=johnSmith"
	secret2 := testSecret.DeepCopy()
	secret2.SetName("testSecret2")
	secret2AnnoMap := map[string]string{
		"kabanero.io/git-0": "https://github.yourdomain1.com",
		"kabanero.io/git-3": "https://github.yourdomain11.com",
		"kabanero.io/git-6": "https://github.yourdomain111.com"}
	secret2.SetAnnotations(secret2AnnoMap)

	secretList := corev1.SecretList{Items: []corev1.Secret{*secret1, *secret2}}
	secret, err := SecretAnnotationFilter(&secretList, hostname, annotationKey)
	if err != nil {
		t.Fatal(fmt.Sprintf("Error retrieving secret associated with hostname: %v and annotation value: %v. Error: %v", hostname, annotationKey, err))
	}
	if secret.Name != secret1.Name {
		t.Fatal(fmt.Sprintf("Secret: %v does not contain an annotation with a value that includes hostname: %v. Expected secret: %v ", secret.Name, hostname, secret1.Name))
	}
}

// Tests that a secret is found when no secrets contain a valid annotation key kabanero.io/git-*, but the
// one of the secrets has an annotation value with the desired hostname.
func TestSecretFilterHostFoundNoKeyFoundInAnnotation(t *testing.T) {
	hostname := "github.mydomain111.com"
	annotationKey := "kabanero.io/git-"
	secret1 := testSecret.DeepCopy()
	secret1AnnoMap := map[string]string{
		"test.io/git-0": "https://github.mydomain1.com/sales",
		"test.io/git-3": "https://github.mydomain11.com/finance/accounting/",
		"test.io/git-6": "https://github.mydomain111.com/procurement/"}
	secret1.SetAnnotations(secret1AnnoMap)

	secret2 := testSecret.DeepCopy()
	secret2.SetName("testSecret2")
	secret2AnnoMap := map[string]string{
		"product.org/git-0": "https://github.yourdomain1.com",
		"product.org/git-3": "https://github.yourdomain11.com",
		"product.org/git-6": "https://github.yourdomain111.com"}
	secret2.SetAnnotations(secret2AnnoMap)

	secretList := corev1.SecretList{Items: []corev1.Secret{*secret1, *secret2}}
	secret, err := SecretAnnotationFilter(&secretList, hostname, annotationKey)
	if err != nil {
		t.Fatal(fmt.Sprintf("Error retrieving secret associated with hostname: %v and annotation value: %v. Error: %v", hostname, annotationKey, err))
	}
	if secret.Name != secret1.Name {
		t.Fatal(fmt.Sprintf("Secret: %v does not contain an annotation with a value that includes hostname: %v. expected secret: %v ", secret.Name, hostname, secret1.Name))
	}
}

// Tests that no error is returned when no secret is found that contains any annotations.
func TestSecretFilterNoAnnotations(t *testing.T) {
	hostname := "github.mydomain1.com"
	annotationKey := "kabanero.io/git-"
	secret1 := testSecret.DeepCopy()
	secret1.SetAnnotations(nil)
	secret2 := testSecret.DeepCopy()
	secret2.SetName("testSecret2")
	secret2.SetAnnotations(nil)

	secretList := corev1.SecretList{Items: []corev1.Secret{*secret1, *secret2}}
	secret, err := SecretAnnotationFilter(&secretList, hostname, annotationKey)
	if err != nil {
		t.Fatal("A secret should not have been found. An error was not expected. Error: ", err)
	}
	if secret != nil {
		t.Fatal("A secret should not have been found. A nil return was expected. Secret: ", secret)
	}
}

// Tests that no error is returned when no secret is found that contains annotation key kabanero.io/git-* or
// annotation value with the desired hostname.
func TestSecretFilterNoHostFoundNoKeyFoundInAnnotation(t *testing.T) {
	hostname := "some.host.that.will.not.be.found.com"
	annotationKey := "kabanero.io/git-"
	secret1 := testSecret.DeepCopy()
	secret2 := testSecret.DeepCopy()
	secret2.SetName("testSecret2")
	secret2AnnoMap := map[string]string{
		"product.org/git-0": "https://github.yourdomain1.com",
		"product.org/git-3": "https://github.yourdomain11.com",
		"product.org/git-6": "https://github.yourdomain111.com"}
	secret2.SetAnnotations(secret2AnnoMap)

	secretList := corev1.SecretList{Items: []corev1.Secret{*secret1, *secret2}}
	secret, err := SecretAnnotationFilter(&secretList, hostname, annotationKey)
	if err != nil {
		t.Fatal("A secret should not have been found. An error was not expected. Error: ", err)
	}
	if secret != nil {
		t.Fatal("A secret should not have been found. A nil return was expected. Secret: ", secret)
	}
}

// Tests that a secret is found when more than one secret contain a valid annotation key kabanero.io/git-*, and
// an annotation value with the desired hostname. The expectation is that the secret with the lowest:
//test.io/git-* is returned.
func TestSecretFilterHostFoundInMultipleSecrets(t *testing.T) {
	secret1 := testSecret.DeepCopy()
	secret1AnnoMap := map[string]string{
		"kabanero.io/git-0": "https://github.mydomain1.com/sales",
		"kabanero.io/git-3": "https://github.mydomain11.com/sales/",
		"test.io/git-6":     "https://github.yourdomain11/procurement/"}
	secret1.SetAnnotations(secret1AnnoMap)

	secret2 := testSecret.DeepCopy()
	secret2.SetName("testSecret2")
	secret2AnnoMap := map[string]string{
		"kabanero.io/git-0": "https://github.yourdomain1.com",
		"kabanero.io/git-3": "https://github.yourdomain11.com",
		"product.org/git-6": "https://github.mydomain11.com/finance/accounting/"}
	secret2.SetAnnotations(secret2AnnoMap)

	secret3 := testSecret.DeepCopy()
	secret3.SetName("testSecret3")
	secret3AnnoMap := map[string]string{
		"kabanero.io/git-0": "https://github.mydomain1.com",
		"kabanero.io/git-4": "https://github.yourdomain11.com",
		"kabanero.io/git-9": "https://github.mydomain11.com/banking/checking/"}
	secret3.SetAnnotations(secret3AnnoMap)

	secret4 := testSecret.DeepCopy()
	secret4.SetName("testSecret4")
	secret4AnnoMap := map[string]string{
		"kabanero.io/git-0": "https://github.mydomain11.com/sales/region?name=ne",
	}
	secret4.SetAnnotations(secret4AnnoMap)

	// Test 1. Random order.
	secretList := corev1.SecretList{Items: []corev1.Secret{*secret1, *secret2, *secret3, *secret4}}
	hostname := "github.mydomain11.com"
	annotationKey := "kabanero.io/git-"
	secret, err := SecretAnnotationFilter(&secretList, hostname, annotationKey)
	if err != nil {
		t.Fatal(fmt.Sprintf("Error retrieving secret associated with hostname: %v and annotation value: %v. Error: %v", hostname, annotationKey, err))
	}
	if secret.Name != secret4.Name {
		t.Fatal(fmt.Sprintf("Secret: %v does not contain an annotation with a value that includes hostname: %v. Expected secret: %v", secret.Name, hostname, secret4.Name))
	}

	// Test 2. Random order.
	hostname = "github.yourdomain11.com"
	secret, err = SecretAnnotationFilter(&secretList, hostname, annotationKey)
	if err != nil {
		t.Fatal(fmt.Sprintf("Error retrieving secret associated with hostname: %v and annotation value: %v. Error: %v", hostname, annotationKey, err))
	}
	if secret.Name != secret2.Name {
		t.Fatal(fmt.Sprintf("Secret: %v does not contain an annotation with a value that includes hostname: %v. Expected secret: %v", secret.Name, hostname, secret2.Name))
	}

	// Test 3. Multiple annotations with exactly "kabanero.io/git-0": "https://github.mydomain1.com/*
	hostname = "github.mydomain1.com"
	secret, err = SecretAnnotationFilter(&secretList, hostname, annotationKey)
	if err != nil {
		t.Fatal(fmt.Sprintf("Error retrieving secret associated with hostname: %v and annotation value: %v. Error: %v", hostname, annotationKey, err))
	}
	if secret.Name != secret1.Name && secret.Name != secret3.Name {
		t.Fatal(fmt.Sprintf("Secret: %v does not contain an annotation with a value that includes hostname: %v. Expected secret: %v or %v", secret.Name, hostname, secret3.Name, secret1.Name))
	}
}
