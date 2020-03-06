package stack

import (
	"testing"

	"github.com/go-logr/logr"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var resolverTestLogger logr.Logger = log.WithValues("Request.Namespace", "test", "Request.Name", "resolver_test")

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

func TestResolveIndex(t *testing.T) {
	repoConfig := kabanerov1alpha2.RepositoryConfig{
		Name: "name",
		Https: kabanerov1alpha2.HttpsProtocolFile{
			Url:                  "https://github.com/kabanero-io/stacks/releases/download/v0.0.1/incubator-index.yaml",
			SkipCertVerification: true,
		},
	}

	index, err := ResolveIndex(nil, repoConfig, "kabanero", []Pipelines{}, []Trigger{}, "", resolverTestLogger)
	if err != nil {
		t.Fatal(err)
	}

	if index == nil {
		t.Fatal("Returned index was nil")
	}

	if index.APIVersion != "v2" {
		t.Fatal("Expected apiVersion == v2")
	}
}

func TestResolveIndexForStacks(t *testing.T) {
	repoConfig := kabanerov1alpha2.RepositoryConfig{
		Name:  "openLibertyTest",
		Https: kabanerov1alpha2.HttpsProtocolFile{Url: "https://github.com/appsody/stacks/releases/download/java-spring-boot2-v0.3.23/incubator-index.yaml"},
	}

	pipelines := []Pipelines{{Id: "testPipeline", Sha256: "1234567890", Url: "https://github.com/kabanero-io/collections/releases/download/0.5.0-rc.2/incubator.common.pipeline.default.tar.gz"}}
	triggers := []Trigger{{Id: "testTrigger", Sha256: "0987654321", Url: "https://github.com/kabanero-io/collections/releases/download/0.5.0-rc.2/incubator.trigger.tar.gz"}}
	index, err := ResolveIndex(nil, repoConfig, "kabanero", pipelines, triggers, "kabanerobeta", resolverTestLogger)

	if err != nil {
		t.Fatal(err)
	}

	if index == nil {
		t.Fatal("The resulting index structure was nil")
	}

	// Validate pipeline entries.
	numStacks := len(index.Stacks)

	if len(index.Stacks[numStacks-numStacks].Pipelines) == 0 {
		t.Fatal("Index.Stacks[0].Pipelines is empty. An entry was expected.")
	}

	c0p0 := index.Stacks[numStacks-numStacks].Pipelines[0]
	if c0p0.Id != "testPipeline" {
		t.Fatal("Expected Index.Stacks[umStacks-numStacks].Pipelines[0] to have a pipeline name of testPipeline. Instead it was: " + c0p0.Id)
	}

	if len(index.Stacks[numStacks-1].Pipelines) == 0 {
		t.Fatal("Index.Stacks[numStacks-1].Pipelines is empty. An entry was expected")
	}

	cLastP0 := index.Stacks[numStacks-1].Pipelines[0]
	if cLastP0.Id != "testPipeline" {
		t.Fatal("Expected Index.Stacks[0].Pipelines[0] to have a pipeline name of testPipeline. Instead it was: " + cLastP0.Id)
	}

	// Validate trigger entry.
	if len(index.Triggers) == 0 {
		t.Fatal("Index.Triggers is empty. An entry was expected")
	}
	trgr := index.Triggers[0]
	if trgr.Id != "testTrigger" {
		t.Fatal("Expected Index.Triggers[0] to have a trigger name of testTrigger. Instead it was: " + trgr.Id)
	}

	// Validate image entry.
	if len(index.Stacks[0].Images) == 0 {
		t.Fatal("index.Stacks[0].Images is empty. An entry was expected")
	}

	image := index.Stacks[0].Images[0]
	if len(image.Image) == 0 {
		t.Fatal("Expected index.Stacks[0].Images[0].Image to have a non-empty value.")
	}

	if len(image.Id) == 0 {
		t.Fatal("Expected index.Stacks[0].Images[0].Id to have a non-empty value.")
	}
}

// Tests that stack index resolution fails if both Git release information Http URL info is not configured in
// the Kabanero CR instance yaml.
func TestResolveIndexForStacksInPublicGitFailure1(t *testing.T) {
	repoConfig := kabanerov1alpha2.RepositoryConfig{
		Name: "openLibertyTest",
	}

	pipelines := []Pipelines{{Id: "testPipeline", Sha256: "1234567890", Url: "https://github.com/kabanero-io/collections/releases/download/0.5.0-rc.2/incubator.common.pipeline.default.tar.gz"}}
	triggers := []Trigger{{Id: "testTrigger", Sha256: "0987654321", Url: "https://github.com/kabanero-io/collections/releases/download/0.5.0-rc.2/incubator.trigger.tar.gz"}}
	index, err := ResolveIndex(nil, repoConfig, "kabanero", pipelines, triggers, "kabanerobeta", resolverTestLogger)

	if err == nil {
		t.Fatal("No Git release or Http url were specified. An error was expected. Index: ", index)
	}
}
func TestSearchStack(t *testing.T) {
	index := &Index{
		APIVersion: "v2",
		Stacks: []Stack{
			Stack{
				DefaultImage:    "java-microprofile",
				DefaultPipeline: "default",
				DefaultTemplate: "default",
				Description:     "Test stack",
				Id:              "java-microprofile",
				Images: []Images{
					Images{},
				},
				Maintainers: []Maintainers{
					Maintainers{},
				},
				Name: "Eclipse Microprofile",
				Pipelines: []Pipelines{
					Pipelines{},
				},
			},
			Stack{
				DefaultImage:    "java-microprofile2",
				DefaultPipeline: "default2",
				DefaultTemplate: "default2",
				Description:     "Test stack 2",
				Id:              "java-microprofile2",
				Images: []Images{
					Images{},
				},
				Maintainers: []Maintainers{
					Maintainers{},
				},
				Name: "Eclipse Microprofile 2",
				Pipelines: []Pipelines{
					Pipelines{},
				},
			},
		},
	}

	stacks, err := SearchStack("java-microprofile2", index)
	if err != nil {
		t.Fatal(err)
	}

	if len(stacks) != 1 {
		t.Fatal("The expected number of stacks is 1, but found: ", len(stacks))
	}

	t.Log(stacks)
}

// Tests that a secret is found when a secret contains a valid annotation key kabanero.io/git-* and the
// annotation value has the desired hostname with no resource path or query string.
func TestSecretFilterHostWithNoExtraPathInAnnotation(t *testing.T) {
	hostname := "github.yourdomain11.com"
	secret1 := testSecret.DeepCopy()
	secret2 := testSecret.DeepCopy()
	secret2.SetName("testSecret2")
	secret2AnnoMap := map[string]string{
		"kabanero.io/git-0": "https://github.yourdomain1.com",
		"kabanero.io/git-3": "https://github.yourdomain11.com",
		"kabanero.io/git-6": "https://github.yourdomain111.com"}
	secret2.SetAnnotations(secret2AnnoMap)

	secretList := corev1.SecretList{Items: []corev1.Secret{*secret1, *secret2}}
	secret, err := secretFilter(&secretList, hostname)
	if err != nil {
		t.Fatal("Error retrieving secret associated with hostname:", hostname, ". Error: ", err)
	}
	if secret.Name != secret2.Name {
		t.Fatal("Secret:", secret.Name, " does not contain an annotation with a value that includes hostname: ", hostname)
	}
}

// Tests that a secret is found when a secret contains a valid annotation key kabanero.io/git-* and the
// annotation value has the desired hostname with a resource path and query string.
func TestSecretFilterHostWithExtraPathInAnnotation(t *testing.T) {
	hostname := "github.mydomain11.com"
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
	secret, err := secretFilter(&secretList, hostname)
	if err != nil {
		t.Fatal("Error retrieving secret associated with hostname:", hostname, ". Error: ", err)
	}
	if secret.Name != secret1.Name {
		t.Fatal("Secret:", secret.Name, " does not contain an annotation with a value that includes hostname: ", hostname)
	}
}

// Tests that a secret is found when no secrets contain a valid annotation key kabanero.io/git-*, but the
// one of the secrets has an annotation value with the desired hostname.
func TestSecretFilterHostFoundNoKeyFoundInAnnotation(t *testing.T) {
	hostname := "github.mydomain111.com"
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
	secret, err := secretFilter(&secretList, hostname)
	if err != nil {
		t.Fatal("Error retrieving secret associated with hostname:", hostname, ". Error: ", err)
	}
	if secret.Name != secret1.Name {
		t.Fatal("Secret:", secret.Name, " does not contain an annotation with a value that includes hostname: ", hostname)
	}
}

// Tests that no error is returned when no secret is found that contains any annotations.
func TestSecretFilterNoAnnotations(t *testing.T) {
	hostname := "github.mydomain1.com"
	secret1 := testSecret.DeepCopy()
	secret1.SetAnnotations(nil)
	secret2 := testSecret.DeepCopy()
	secret2.SetName("testSecret2")
	secret2.SetAnnotations(nil)

	secretList := corev1.SecretList{Items: []corev1.Secret{*secret1, *secret2}}
	secret, err := secretFilter(&secretList, hostname)
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
	secret1 := testSecret.DeepCopy()
	secret2 := testSecret.DeepCopy()
	secret2.SetName("testSecret2")
	secret2AnnoMap := map[string]string{
		"product.org/git-0": "https://github.yourdomain1.com",
		"product.org/git-3": "https://github.yourdomain11.com",
		"product.org/git-6": "https://github.yourdomain111.com"}
	secret2.SetAnnotations(secret2AnnoMap)

	secretList := corev1.SecretList{Items: []corev1.Secret{*secret1, *secret2}}
	secret, err := secretFilter(&secretList, hostname)
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
	secret, err := secretFilter(&secretList, hostname)
	if err != nil {
		t.Fatal("Error retrieving secret associated with hostname:", hostname, ". Error: ", err)
	}
	if secret.Name != secret4.Name {
		t.Fatal("Secret:", secret.Name, " does not contain an annotation with a value that includes hostname: ", hostname, ". Expected secret: ", secret4.Name)
	}

	// Test 2. Random order.
	hostname = "github.yourdomain11.com"
	secret, err = secretFilter(&secretList, hostname)
	if err != nil {
		t.Fatal("Error retrieving secret associated with hostname:", hostname, ". Error: ", err)
	}
	if secret.Name != secret2.Name {
		t.Fatal("Secret:", secret.Name, " does not contain an annotation with a value that includes hostname: ", hostname, ". Expected secret: ", secret2.Name)
	}

	// Test 3. Multiple annotations with exactly "kabanero.io/git-0": "https://github.mydomain1.com/*
	hostname = "github.mydomain1.com"
	secret, err = secretFilter(&secretList, hostname)
	if err != nil {
		t.Fatal("Error retrieving secret associated with hostname:", hostname, ". Error: ", err)
	}
	if secret.Name != secret1.Name && secret.Name != secret3.Name {
		t.Fatal("Secret:", secret.Name, " does not contain an annotation with a value that includes hostname:", hostname, ". Expected secrets: ", secret1.Name, "  or ", secret3.Name)
	}
}
