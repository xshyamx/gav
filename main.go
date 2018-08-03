package main

import (
	"crypto/sha1"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var (
	debug        bool
	outFile      string
	dependencies chan Dependency

//  wg           sync.WaitGroup
)

const pomTemplate = `<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
  <modelVersion>4.0.0</modelVersion>

  <groupId>group-id</groupId>
  <artifactId>artifact-id</artifactId>
  <version>1.0</version>
  <packaging>jar</packaging>

  <properties>
    <project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>
  </properties>

  <dependencies>{{range .}}
    <dependency>
      <groupId>{{.GroupId}}</groupId>
      <artifactId>{{.ArtifactId}}</artifactId>
      <version>{{.Version}}</version>
    </dependency>{{end}}
  </dependencies>
</project>`

type Dependency struct {
	GroupId    string `json:"groupId"`
	ArtifactId string `json:"artifactId"`
	Version    string `json:"version"`
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
	flag.BoolVar(&debug, "d", false, "Print debug messages")
	flag.StringVar(&outFile, "o", "pom.xml", "Output file name")
	flag.Parse()
	nDirs := flag.NArg()
	if nDirs == 0 {
		flag.PrintDefaults()
	}
	debugf("debug: %t, outFile: %s, dirs: %d", debug, outFile, nDirs)
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
	pomTmpl := template.Must(template.New("pom").Parse(pomTemplate))
	out, err := os.Create(outFile)
	if err != nil {
		debugf("Failed to open file %s", outFile)
		panic(err)
	}
	err = pomTmpl.Execute(out, deps)
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
