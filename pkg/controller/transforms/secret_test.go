package transforms

import (
	"fmt"
	"strings"
	"testing"
)

func TestMountSecret(t *testing.T) {
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
  name: kabanero-events
spec:
  replicas: 1
  selector:
    matchLabels:
      name: kabanero-events
  template:
    metadata:
      labels:
        name: kabanero-events
    spec:
      serviceAccountName: kabanero-events
      containers:
        - name: kabanero-events
          image: image
          imagePullPolicy: Always
          env:
            - name: KUBE_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
          volumeMounts:
      volumes:`,

			expectedOutput: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: kabanero-events
spec:
  replicas: 1
  selector:
    matchLabels:
      name: kabanero-events
  template:
    metadata:
      labels:
        name: kabanero-events
    spec:
      containers:
      - env:
        - name: KUBE_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        image: image
        imagePullPolicy: Always
        name: kabanero-events
        volumeMounts:
        - mountPath: /etc/tls
          name: kabanero-events-serving-cert
          readOnly: true
      serviceAccountName: kabanero-events
      volumes:
      - name: kabanero-events-serving-cert
        secret:
          secretName: kabanero-events-serving-cert`,
		},
		{
			name: "deployment-novolume",
			inputYaml: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: kabanero-events
spec:
  replicas: 1
  selector:
    matchLabels:
      name: kabanero-events
  template:
    metadata:
      labels:
        name: kabanero-events
    spec:
      serviceAccountName: kabanero-events
      containers:
        - name: kabanero-events
          image: image
          imagePullPolicy: Always
          env:
            - name: KUBE_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace`,
			expectedOutput: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: kabanero-events
spec:
  replicas: 1
  selector:
    matchLabels:
      name: kabanero-events
  template:
    metadata:
      labels:
        name: kabanero-events
    spec:
      containers:
      - env:
        - name: KUBE_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        image: image
        imagePullPolicy: Always
        name: kabanero-events
        volumeMounts:
        - mountPath: /etc/tls
          name: kabanero-events-serving-cert
          readOnly: true
      serviceAccountName: kabanero-events
      volumes:
      - name: kabanero-events-serving-cert
        secret:
          secretName: kabanero-events-serving-cert`,
		}}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s", tc.name), func(t *testing.T) {
			u, err := unmarshal([]byte(tc.inputYaml))
			if err != nil {
				t.Fatal(err)
			}
			deployment := &u[0]
			err = MountSecret("kabanero-events-serving-cert", "/etc/tls")(deployment)
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
