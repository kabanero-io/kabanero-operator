package client

import (
	"github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	"io/ioutil"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"reflect"
	"strings"
	"testing"
)

func TestUnmarshal(t *testing.T) {
	v1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme)

	datas := []struct {
		yaml string
		t    interface{}
	}{
		{
			`apiVersion: v1
kind: Namespace
metadata:
  name: myobject`,
			&v1.Namespace{},
		},
		{
			`apiVersion: kabanero.io/v1alpha1
kind: Kabanero
metadata:
  name: myobject`,
			&v1alpha1.Kabanero{},
		},
		{
			`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{},"name":"myobject","namespace":"default"},"spec":{"replicas":2,"selector":{"matchLabels":{"app":"nginx"}},"template":{"metadata":{"labels":{"app":"nginx"}},"spec":{"containers":[{"image":"nginx:1.7.9","name":"nginx","ports":[{"containerPort":80}]}]}}}}`,
			&appsv1.Deployment{},
		},
	}

	for _, data := range datas {
		c := NewClient(&Config{})
		r := ioutil.NopCloser(strings.NewReader(data.yaml))
		objs, err := c.Unmarshal(r, "yaml")
		if err != nil {
			t.Fatal(err)
		}

		if len(objs) != 1 {
			t.Fatalf("Expected 1 object but found %v", len(objs))
		}

		obj := objs[0]

		if reflect.TypeOf(obj) != reflect.TypeOf(data.t) {
			t.Fatal("unpexected types", reflect.TypeOf(obj), reflect.TypeOf(data.t))
		}

		if obj.GetName() != "myobject" {
			t.Fatalf("Expected object to have name 'myobject', found %v", obj.GetName())
		}
	}
}
