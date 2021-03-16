package main

import (
	"crypto/sha1"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var (
	debug        bool
	outFile      string
	dependencies chan Dependency

	//  wg           sync.WaitGroup
)

//go:embed pom-template.xml
var pomTemplate string

type Dependency struct {
	GroupId    string `json:"groupId"`
	ArtifactId string `json:"artifactId"`
	Version    string `json:"version"`
}

type POM struct {
	Project      Dependency
	Dependencies []Dependency
}

type Result struct {
	Path       string      `json:"path"`
	Filename   string      `json:"filename"`
	Hash       string      `json:"sha1"`
	Dependency *Dependency `json:"dependency,omitempty"`
}

type RepoFunc func(string) (Dependency, error)

var repoFuncs = []RepoFunc{
	central,
	jboss,
	spring,
	jfrog,
}

// print debug messages to console
func debugf(format string, args ...interface{}) {
	if debug {
		fmt.Println("[DEBUG] " + fmt.Sprintf(format, args...))
	}
}

func getHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func main() {
	pomDep := Dependency{
		GroupId:    "com.example",
		ArtifactId: "sample",
		Version:    "1.0.0",
	}
	pwd, err := os.Getwd()
	if err == nil {
		pomDep.ArtifactId = filepath.Base(pwd)
	}
	flag.BoolVar(&debug, "d", false, "Print debug messages")
	flag.StringVar(&outFile, "o", "pom.xml", "Output file name")
	flag.StringVar(&pomDep.GroupId, "g", pomDep.GroupId, "Optional groupId for the generated pom.xml")
	flag.StringVar(&pomDep.ArtifactId, "a", pomDep.ArtifactId, "Optional artifactId for the generated pom.xml")
	flag.StringVar(&pomDep.Version, "v", pomDep.Version, "Optional version for the generated pom.xml")
	flag.Parse()
	nDirs := flag.NArg()
	if nDirs == 0 {
		flag.PrintDefaults()
		return
	}
	debugf("debug: %t, outFile: %s, dirs: %d", debug, outFile, nDirs)
	debugf("debug: %+v", pomDep)
	var deps []Dependency
	var results []Result
	dirs := flag.Args()
	var found, total = 0, 0
	for _, dir := range dirs {
		debugf("Scanning %s", dir)
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				// file
				if strings.HasSuffix(path, ".jar") {
					hash, err := getHash(path)
					if err != nil {
						return err
					}
					relPath, _ := filepath.Rel(dir, path)
					result := Result{
						Path:     relPath,
						Filename: filepath.Base(path),
						Hash:     hash,
					}
					debugf("%s <- %s", result.Hash, result.Filename)
					total++
					for _, repoFunc := range repoFuncs {
						if dep, err := repoFunc(result.Hash); err != nil {
							// failed one
							result.Dependency = nil
							debugf("Failed to find dependency from %v", err)
						} else {
							debugf("[%s] %s -> %+v, %v", hash, filepath.Base(path), dep, err)
							result.Dependency = &dep
							deps = append(deps, dep)
							found++
							break
						}
					}
					results = append(results, result)
				}
			}
			return nil
		})
	}
	if len(deps) == 0 {
		fmt.Printf("No dependencies identified. Will not write %s\n", outFile)
		return
	}
	pomTmpl := template.Must(template.New("pom").Parse(pomTemplate))
	out, err := os.Create(outFile)
	if err != nil {
		debugf("Failed to open file %s", outFile)
		panic(err)
	}
	err = pomTmpl.Execute(out, POM{
		Project:      pomDep,
		Dependencies: deps,
	})
	if err == nil {
		fmt.Printf("%d dependencies out of %d jars\n", found, total)
	}
	if debug {
		buf, err := json.MarshalIndent(struct {
			BaseDirs []string `json:"basedirs"`
			Results  []Result `json:"results"`
		}{
			dirs,
			results,
		}, "", "  ")
		if err != nil {
			debugf("Failed to marshal debug results json")
			// don't panic
		}
		if err = ioutil.WriteFile("debug.json", buf, 0655); err != nil {
			debugf("Failed to write debug.json")
		}
	}
}
