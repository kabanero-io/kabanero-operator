package client

import (
	"strings"
	"testing"
)

func TestApply(t *testing.T) {
	yamlText := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: echo
  namespace: default
  labels:
    app: echo
spec:
  replicas: 1
  selector:
    matchLabels:
      app: echo
  template:
    metadata:
      labels:
        app: echo
    spec:
      hostname: "myhost"
      containers:
        - name: echo
          image: "ubuntu:2"
          imagePullPolicy: IfNotPresent`

	r := strings.NewReader(yamlText)
	_, err := DefaultClient.ApplyText(r, nil)
	if err != nil {
		t.Fatal(err)
	}

}
