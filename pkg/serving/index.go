package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/elsony/devfile2-registry/tools/types"
	"gopkg.in/yaml.v2"
)

// genIndex generate new index from meta.yaml files in dir.
// meta.yaml file is expected to be in dir/<devfiledir>/meta.yaml
func genIndex(dir string) ([]types.MetaIndex, error) {

	var index []types.MetaIndex

	dirs, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range dirs {
		if file.IsDir() {
			var meta types.Meta
			metaFile, err := ioutil.ReadFile(filepath.Join(dir, file.Name(), "meta.yaml"))
			if err != nil {
				return nil, err
			}
			err = yaml.Unmarshal(metaFile, &meta)
			if err != nil {
				return nil, err
			}

			self := fmt.Sprintf("/%s/%s/%s", filepath.Base(dir), file.Name(), "devfile.yaml")

			metaIndex := types.MetaIndex{
				Meta: meta,
				Links: types.Links{
					Self: self,
				},
			}
			index = append(index, metaIndex)
		}
	}
	return index, nil
}

func main() {
	devfiles := flag.String("devfiles-dir", "", "Directory containing devfiles.")
	output := flag.String("index", "", "Index filaname. This is where the index in JSON format will be saved.")

	flag.Parse()

	if *devfiles == "" {
		log.Fatal("Provide devfile directory.")
	}

	if *output == "" {
		log.Fatal("Provide index file.")
	}

	index, err := genIndex(*devfiles)
	if err != nil {
		log.Fatal(err)
	}
	b, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		log.Fatal(err)

	}
	err = ioutil.WriteFile(*output, b, 0644)
	if err != nil {
		log.Fatal(err)

	}
}
