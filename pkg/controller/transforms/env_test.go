package transforms

import (
	"bytes"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
	"strings"
	"testing"
)

func unmarshal(b []byte) ([]unstructured.Unstructured, error) {
	decoder := yaml.NewYAMLToJSONDecoder(bytes.NewReader(b))
	objs := []unstructured.Unstructured{}
	var err error
	for {
		out := unstructured.Unstructured{}
		err = decoder.Decode(&out)
		if err != nil {
			break
		}
		if len(out.Object) == 0 {
			continue
		}
		objs = append(objs, out)
	}
	if err != io.EOF {
		return nil, err
	}
	return objs, nil
}

func marshal(u *unstructured.Unstructured) ([]byte, error) {
	var o bytes.Buffer
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	err := s.Encode(u, &o)
	if err != nil {
		return nil, err
	}
	b := o.Bytes()
	return b, nil
}

func TestReplaceEnvVariable(t *testing.T) {
	tests := []struct {
		name           string
		inputYaml      string
		expectedOutput string
		expectedError  string
	}{
		{
			name: "no matches",
			inputYaml: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: mydeployment
spec: {}`,
			expectedOutput: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: mydeployment
spec: {}`,
		},
		{
			name: "deployment",
			inputYaml: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: appsody-operator
spec:
  replicas: 1
  selector:
    matchLabels:
      name: appsody-operator
  template:
    metadata:
      labels:
        name: appsody-operator
    spec:
      serviceAccountName: appsody-operator
      containers:
        - name: appsody-operator
          image: image
          command:
          - appsody-operator
          imagePullPolicy: Always
          env:
            - name: WATCH_NAMESPACE
              valueFrom: watch
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: OPERATOR_NAME
              value: "appsody-operator"`,

			expectedOutput: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: appsody-operator
spec:
  replicas: 1
  selector:
    matchLabels:
      name: appsody-operator
  template:
    metadata:
      labels:
        name: appsody-operator
    spec:
      containers:
      - command:
        - appsody-operator
        env:
        - name: WATCH_NAMESPACE
          value: mynamespace
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: OPERATOR_NAME
          value: appsody-operator
        image: image
        imagePullPolicy: Always
        name: appsody-operator
      serviceAccountName: appsody-operator`,
		}}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s", tc.name), func(t *testing.T) {
			u, err := unmarshal([]byte(tc.inputYaml))
			if err != nil {
				t.Fatal(err)
			}
			deployment := &u[0]
			err = ReplaceEnvVariable("WATCH_NAMESPACE", "mynamespace")(deployment)
			if err != nil && tc.expectedError != "" && tc.expectedError == err.Error() {
				//Matches expected error
			} else if err != nil && tc.expectedError != "" {
				t.Fatalf("Expected error `%v` but found error `%v`", tc.expectedError, err.Error())
			} else if err != nil {
				t.Fatal(err)
			} else {
				b, err := marshal(deployment)
				if err != nil {
					t.Fatal(err)
				}
				if strings.TrimSpace(tc.expectedOutput) != strings.TrimSpace(string(b)) {
					t.Log("Expected: ", tc.expectedOutput)
					t.Log("Found: ", string(b))

					t.Fatal("Expected output did not match")
				}
			}
		})
	}
}

func TestAddEnvVariable(t *testing.T) {
	tests := []struct {
		name           string
		inputYaml      string
		expectedOutput string
		expectedError  string
	}{
		{
			name: "no matches",
			inputYaml: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: mydeployment
spec: {}`,
	    expectedError: "No containers entry in deployment spec: &{map[apiVersion:apps/v1 kind:Deployment metadata:map[name:mydeployment] spec:map[]]}",
		},
		{
			name: "deployment",
			inputYaml: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: appsody-operator
spec:
  replicas: 1
  selector:
    matchLabels:
      name: appsody-operator
  template:
    metadata:
      labels:
        name: appsody-operator
    spec:
      serviceAccountName: appsody-operator
      containers:
        - name: appsody-operator
          image: image
          command:
          - appsody-operator
          imagePullPolicy: Always
          env:
            - name: WATCH_NAMESPACE
              valueFrom: watch
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: OPERATOR_NAME
              value: "appsody-operator"`,

			expectedOutput: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: appsody-operator
spec:
  replicas: 1
  selector:
    matchLabels:
      name: appsody-operator
  template:
    metadata:
      labels:
        name: appsody-operator
    spec:
      containers:
      - command:
        - appsody-operator
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: OPERATOR_NAME
          value: appsody-operator
        - name: WATCH_NAMESPACE
          value: mynamespace
        image: image
        imagePullPolicy: Always
        name: appsody-operator
      serviceAccountName: appsody-operator`,
		},
		{
			name: "deployment-newvar",
			inputYaml: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: appsody-operator
spec:
  replicas: 1
  selector:
    matchLabels:
      name: appsody-operator
  template:
    metadata:
      labels:
        name: appsody-operator
    spec:
      serviceAccountName: appsody-operator
      containers:
        - name: appsody-operator
          image: image
          command:
          - appsody-operator
          imagePullPolicy: Always
          env:
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: OPERATOR_NAME
              value: "appsody-operator"`,

			expectedOutput: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: appsody-operator
spec:
  replicas: 1
  selector:
    matchLabels:
      name: appsody-operator
  template:
    metadata:
      labels:
        name: appsody-operator
    spec:
      containers:
      - command:
        - appsody-operator
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: OPERATOR_NAME
          value: appsody-operator
        - name: WATCH_NAMESPACE
          value: mynamespace
        image: image
        imagePullPolicy: Always
        name: appsody-operator
      serviceAccountName: appsody-operator`,
		}}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s", tc.name), func(t *testing.T) {
			u, err := unmarshal([]byte(tc.inputYaml))
			if err != nil {
				t.Fatal(err)
			}
			deployment := &u[0]
			err = AddEnvVariable("WATCH_NAMESPACE", "mynamespace")(deployment)
			if err != nil && tc.expectedError != "" && tc.expectedError == err.Error() {
				//Matches expected error
			} else if err != nil && tc.expectedError != "" {
				t.Fatalf("Expected error `%v` but found error `%v`", tc.expectedError, err.Error())
			} else if err != nil {
				t.Fatal(err)
			} else {
				b, err := marshal(deployment)
				if err != nil {
					t.Fatal(err)
				}
				if strings.TrimSpace(tc.expectedOutput) != strings.TrimSpace(string(b)) {
					t.Log("Expected: ", tc.expectedOutput)
					t.Log("Found: ", string(b))

					t.Fatal("Expected output did not match")
				}
			}
		})
	}
}
