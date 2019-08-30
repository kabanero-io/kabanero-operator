package collection

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"net/http"
	"strings"
)

func DownloadToByte(url string) ([]byte, error) {
	r, err := http.Get(url)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not download file: %v", url))
	}
	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
	return b, err
}

//Read the manifests from a tar.gz archive
//It would be better to use the manifest.yaml as the index, and check the signatures
//For now, ignore manifest.yaml and return all other yaml files from the archive
func DecodeManifests(archive []byte, collectionName string) ([]unstructured.Unstructured, error) {
	manifests := []unstructured.Unstructured{}

	r := bytes.NewReader(archive)
	gzReader, err := gzip.NewReader(r)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not read manifest gzip"))
	}
	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, errors.New(fmt.Sprintf("Could not read manifest tar"))
		}

		//For now skip manifest.yaml, rather than utilizing it as the index of the archive
		switch {
		case header.Name == "./manifest.yaml":
			break
		case strings.HasSuffix(header.Name, ".yaml"):
			//Buffer the document for further processing
			b := make([]byte, header.Size)
			_, err := tarReader.Read(b)

			//Apply the Kabanero yaml directive processor
			s := &DirectiveProcessor{}
			b, err = s.Render(b, map[string]interface{}{"CollectionName": collectionName})
			if err != nil {
				return nil, err
			}

			decoder := yaml.NewYAMLToJSONDecoder(bytes.NewReader(b))
			out := unstructured.Unstructured{}
			err = decoder.Decode(&out)
			if err != nil {
				fmt.Sprintf("Error decoding %v", header.Name)
			}
			manifests = append(manifests, out)
		}
	}
	return manifests, nil
}

func GetManifests(url string, collectionName string) ([]unstructured.Unstructured, error) {
	b, err := DownloadToByte(url)
	if err != nil {
		return nil, err
	}

	manifests, err := DecodeManifests(b, collectionName)
	if err != nil {
		return nil, err
	}
	return manifests, err
}
