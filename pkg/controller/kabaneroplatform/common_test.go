package kabaneroplatform

import (
	"fmt"
	"strings"
	"testing"

	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	"github.com/kabanero-io/kabanero-operator/pkg/assets/config"
	"github.com/kabanero-io/kabanero-operator/pkg/versioning"
)

func TestImageUriWithOverrides(t *testing.T) {
	tests := []struct {
		name               string
		revision           versioning.SoftwareRevision
		repositoryOverride string
		tagOverride        string
		imageOverride      string
		expectedImage      string
		expectedError      string
	}{
		{
			name:               "no-overrides",
			revision:           versioning.SoftwareRevision{Identifiers: map[string]interface{}{versioning.REPOSITORY_IDENTIFIER: "repository", versioning.TAG_IDENTIFIER: "tag"}},
			repositoryOverride: "",
			tagOverride:        "",
			imageOverride:      "",
			expectedImage:      "repository:tag",
		},
		{
			name:               "override-tag",
			revision:           versioning.SoftwareRevision{Identifiers: map[string]interface{}{versioning.REPOSITORY_IDENTIFIER: "repository", versioning.TAG_IDENTIFIER: "tag"}},
			repositoryOverride: "",
			tagOverride:        "overriddenT",
			imageOverride:      "",
			expectedImage:      "repository:overriddenT",
		},
		{
			name:               "override-repository",
			revision:           versioning.SoftwareRevision{Identifiers: map[string]interface{}{versioning.REPOSITORY_IDENTIFIER: "repository", versioning.TAG_IDENTIFIER: "tag"}},
			repositoryOverride: "overriddenR",
			tagOverride:        "",
			imageOverride:      "",
			expectedImage:      "overriddenR:tag",
		},
		{
			name:               "override repository and tag",
			revision:           versioning.SoftwareRevision{Identifiers: map[string]interface{}{versioning.REPOSITORY_IDENTIFIER: "repository", versioning.TAG_IDENTIFIER: "tag"}},
			repositoryOverride: "overriddenR",
			tagOverride:        "overriddenT",
			imageOverride:      "",
			expectedImage:      "overriddenR:overriddenT",
		},
		{
			name:               "ambiguous-override",
			revision:           versioning.SoftwareRevision{Identifiers: map[string]interface{}{versioning.REPOSITORY_IDENTIFIER: "repository", versioning.TAG_IDENTIFIER: "tag"}},
			repositoryOverride: "overridden-repo",
			tagOverride:        "overridden-tag",
			imageOverride:      "overridden-image:image-tag",
			expectedImage:      "overridden-image:image-tag",
		},
		{
			name:          "key type error",
			revision:      versioning.SoftwareRevision{Identifiers: map[string]interface{}{versioning.REPOSITORY_IDENTIFIER: struct{}{}, versioning.TAG_IDENTIFIER: "tag"}},
			expectedError: "The embedded identifier `repository` was expected to be a string",
		},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s", tc.name), func(t *testing.T) {
			i, err := imageUriWithOverrides(tc.repositoryOverride, tc.tagOverride, tc.imageOverride, tc.revision)
			if err != nil && tc.expectedError == err.Error() {
				//Error matches expected error, pass
			} else if err != nil {
				t.Fatal("Unexpected error: ", err)
			} else if i != tc.expectedImage {
				t.Fatalf("Image `%v` does not match expected `%v`", i, tc.expectedImage)
			} else {
				//Matches expectation
			}
		})
	}
}

func TestRenderOrchestration(t *testing.T) {
	tests := []struct {
		name                   string
		filename               string
		context                map[string]interface{}
		expectedResultContains string
		expectedError          error
	}{
		{
			name:                   "default",
			filename:               "orchestrations/che/0.1/codewind-che-cr.yaml",
			context:                map[string]interface{}{"repository": "myimage"},
			expectedResultContains: "cheImage: myimage",
		},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s", tc.name), func(t *testing.T) {
			r, err := config.Open(tc.filename)
			if err != nil {
				t.Fatal("Unexpected error: ", err)
			}
			result, err := renderOrchestration(r, tc.context)
			if err != nil && tc.expectedError != err {
				t.Fatal("Unexpected error: ", err)
			} else if !strings.Contains(result, tc.expectedResultContains) {
				t.Fatalf("Expected `%v` but found `%v`", tc.expectedResultContains, result)
			}
		})
	}
}

func TestResolveSoftwareRevision(t *testing.T) {
	tests := []struct {
		name                   string
		filename               string
		context                map[string]interface{}
		expectedResultContains string
		expectedError          error
	}{
		{
			name: "default",
		},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s", tc.name), func(t *testing.T) {
			k := &kabanerov1alpha1.Kabanero{
				Spec: kabanerov1alpha1.KabaneroSpec{},
			}
			rev, err := resolveSoftwareRevision(k, "cli-services", "")
			_ = rev
			if err != nil {
				t.Fatal("Unexpected error: ", err)
			}
		})
	}
}
