// +build ignore

package main

import (
	"log"
	"net/http"
	"path"
	"path/filepath"
	"runtime"

	"github.com/shurcooL/vfsgen"
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	cwd := path.Dir(filename)
	templates := http.Dir(filepath.Join(cwd, "../../config"))
	if err := vfsgen.Generate(templates, vfsgen.Options{
		Filename:     "config/config_vfsdata.go",
		PackageName:  "config",
		BuildTags:    "!dev",
		VariableName: "assets",
	}); err != nil {
		log.Fatalln(err)
	}
}
