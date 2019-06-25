package client

import (
	"io"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func Marshal(o runtime.Object, w io.Writer, format string) error {
	var s *json.Serializer
	if format == "json" {
		s = json.NewSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme, false)
	} else if format == "yaml" {
		s = json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	}

	err := s.Encode(o, w)
	if err != nil {
		return err
	}

	return nil
}
