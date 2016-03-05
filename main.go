package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

type dependency struct {
	ImportPath string
	Comment    string `json:",omitempty"`
	Rev        string // VCS-specific commit ID.
}

type Godeps struct {
	ImportPath string
	GoVersion  string   // Abridged output of 'go version'.
	Packages   []string // Arguments to godep save, if any.
	Deps       []*dependency
}

// How to use:
//   mergegodep -od ..kubernetes/Godeps -nd ..storage/etcd/Godeps

func main() {
	od := flag.String("od", "", "old godep")
	nd := flag.String("nd", "", "new godep")
	flag.Parse()

	if *od == "" || *nd == "" {
		log.Println("please set both -od and -nd")
		return
	}

	of, err := ioutil.ReadFile(filepath.Join(*od, "Godeps.json"))
	if err != nil {
		panic(err)
	}
	og := new(Godeps)
	err = json.Unmarshal(of, og)
	if err != nil {
		panic(err)
	}

	nf, err := ioutil.ReadFile(filepath.Join(*nd, "Godeps.json"))
	if err != nil {
		panic(err)
	}
	ng := new(Godeps)
	err = json.Unmarshal(nf, ng)
	if err != nil {
		panic(err)
	}

	pathToDep := make(map[string]*dependency)
	for _, dep := range og.Deps {
		pathToDep[dep.ImportPath] = dep
	}

	// update in new
	for _, dep := range ng.Deps {
		oldDep, ok := pathToDep[dep.ImportPath]
		if ok && dep.Rev == oldDep.Rev {
			continue
		}

		pathToDep[dep.ImportPath] = dep

		srcFolder := filepath.Join(*nd, "_workspace/src", dep.ImportPath)
		dstFolder := filepath.Join(*od, "_workspace/src", dep.ImportPath)

		if ok {
			log.Printf("rm %s/*", dstFolder)
			files, err := ioutil.ReadDir(dstFolder)
			if err != nil {
				panic(err)
			}

			for _, f := range files {
				if f.IsDir() {
					continue
				}
				err = os.Remove(filepath.Join(dstFolder, f.Name()))
				if err != nil {
					panic(err)
				}
			}
		}

		log.Printf("cp -rf %s/ %s/", srcFolder, dstFolder)
		err = exec.Command("cp", "-rf", srcFolder+"/", dstFolder+"/").Run()
		if err != nil {
			panic(err)
		}
	}

	paths := make([]string, 0, len(pathToDep))
	for path := range pathToDep {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	deps := make([]*dependency, 0, len(paths))
	for _, path := range paths {
		deps = append(deps, pathToDep[path])
	}

	output := &Godeps{
		ImportPath: og.ImportPath,
		GoVersion:  og.GoVersion,
		Packages:   og.Packages,
		Deps:       deps,
	}
	data, err := json.MarshalIndent(output, "", "\t")
	if err != nil {
		panic(err)
	}
	data = append(data, '\n')
	err = ioutil.WriteFile(filepath.Join(*od, "Godeps.json"), data, 0644)
	if err != nil {
		panic(err)
	}

	log.Printf("done!")
}
