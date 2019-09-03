package collection

import (
	"strings"
	"testing"
)

func TestDirectiveProcessor(t *testing.T) {
	b := []byte(`
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
    `)

	expected := []byte(`
apiVersion: tekton.dev/v1alpha1
kind: Pipeline
metadata:
  name: MyCollection-build-deploy-pipeline
spec:
  resources:
  - resource1
  tasks:
    - name: build-task
      taskRef:
        name: MyCollection-build-task
    `)

	context := map[string]interface{}{"CollectionName": "MyCollection"}

	r := &DirectiveProcessor{}
	b_output, err := r.Render(b, context)

	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(b_output)) != strings.TrimSpace(string(expected)) {
		t.Fatal("Output did not match expectations", string(b_output), string(expected))
	}
}
