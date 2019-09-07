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
func DecodeManifests(archive []byte, renderingContext map[string]interface{}) ([]unstructured.Unstructured, error) {
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
		case header.Name == "manifest.yaml" || strings.HasSuffix(header.Name, "/manifest.yaml"):
			break
		case strings.HasSuffix(header.Name, ".yaml"):
			//Buffer the document for further processing
			b := make([]byte, header.Size)
			i, err := tarReader.Read(b)
			//An EOF error is normal, as long as bytes read > 0
			if err == io.EOF && i == 0 || err != nil && err != io.EOF {
				return nil, fmt.Errorf("Error reading archive %v: %v", header.Name, err.Error())
			}

			//Apply the Kabanero yaml directive processor
			s := &DirectiveProcessor{}
			b, err = s.Render(b, renderingContext)
			if err != nil {
				return nil, fmt.Errorf("Error processing directives %v: %v", header.Name, err.Error())
			}

			decoder := yaml.NewYAMLToJSONDecoder(bytes.NewReader(b))
			out := unstructured.Unstructured{}
			err = decoder.Decode(&out)
			if err != nil {
				return nil, fmt.Errorf("Error decoding %v: %v", header.Name, err.Error())
			}
			manifests = append(manifests, out)
		}
	}
	return manifests, nil
}

func GetManifests(url string, renderingContext map[string]interface{}) ([]unstructured.Unstructured, error) {
	b, err := DownloadToByte(url)
	if err != nil {
		return nil, err
	}

	manifests, err := DecodeManifests(b, renderingContext)
	if err != nil {
		return nil, err
	}
	return manifests, err
}
