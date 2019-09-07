package collection

import (
	"fmt"
	"strings"
	"testing"
)

func TestDirectiveProcessor(t *testing.T) {
	tests := []struct {
		name     string
		provided []byte
		expected []byte
	}{{
		name: "Substitute CollectionName",
		provided: []byte(`
#Kabanero! on activate substitute CollectionName for text '${collection-name}'
apiVersion: tekton.dev/v1alpha1
kind: Pipeline
metadata:
  name: ${collection-name}-build-deploy-pipeline
spec:
  resources:
  - resource1
  tasks:
    - name: build-task
      taskRef:
        name: ${collection-name}-build-task
    `),

		expected: []byte(`
apiVersion: tekton.dev/v1alpha1
kind: Pipeline
metadata:
  name: My Collection Name-build-deploy-pipeline
spec:
  resources:
  - resource1
  tasks:
    - name: build-task
      taskRef:
        name: My Collection Name-build-task
    `),
	},
		{
			name: "Substitute CollectionId",
			provided: []byte(`
#Kabanero! on activate substitute CollectionId for text 'CollectionId'
apiVersion: tekton.dev/v1alpha1
kind: Pipeline
metadata:
  name: CollectionId-build-deploy-pipeline
spec:
  resources:
  - resource1
  tasks:
    - name: build-task
      taskRef:
        name: CollectionId-build-task
    `),

			expected: []byte(`
apiVersion: tekton.dev/v1alpha1
kind: Pipeline
metadata:
  name: my-collection-build-deploy-pipeline
spec:
  resources:
  - resource1
  tasks:
    - name: build-task
      taskRef:
        name: my-collection-build-task
    `),
		}}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s", tc.name), func(t *testing.T) {
			context := map[string]interface{}{
				"CollectionName": "My Collection Name",
				"CollectionId":   "my-collection",
			}

			r := &DirectiveProcessor{}
			b_output, err := r.Render(tc.provided, context)

			if err != nil {
				t.Fatal(err)
			}
			if strings.TrimSpace(string(b_output)) != strings.TrimSpace(string(tc.expected)) {
				t.Fatal("Output did not match expectations", string(b_output), string(tc.expected))
			}
		})
	}
}
