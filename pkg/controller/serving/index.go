package serving

import (
	//"encoding/json"
	//"flag"
	// "fmt"
	"io/ioutil"
	//"log"
	"path/filepath"

	"github.com/elsony/devfile2-registry/tools/types"
	"gopkg.in/yaml.v2"
)

// genIndex generate new index from meta.yaml files in dir.
// meta.yaml file is expected to be in dir/<stack>/<version>/meta.yaml
func genIndex(dir string) ([]types.MetaIndex, error) {

	var index []types.MetaIndex

	stackdirs, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, stackdir := range stackdirs {
		if stackdir.IsDir() {
			versiondirs, err := ioutil.ReadDir(filepath.Join(dir, stackdir.Name()))
			if err != nil {
				return nil, err
			}
			for _, versiondir := range versiondirs {
				var meta types.Meta
				metaFile, err := ioutil.ReadFile(filepath.Join(dir, stackdir.Name(), versiondir.Name(), "meta.yaml"))
				if err != nil {
					return nil, err
				}
				err = yaml.Unmarshal(metaFile, &meta)
				if err != nil {
					return nil, err
				}

				self := filepath.Join(stackdir.Name(), versiondir.Name(), "devfile.yaml")

				metaIndex := types.MetaIndex{
					Meta: meta,
					Links: types.Links{
						Self: self,
					},
				}
				index = append(index, metaIndex)
			}
		}
	}
	return index, nil
}

