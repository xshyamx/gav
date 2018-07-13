package main

import (
  "bytes"
  "crypto/sha1"
  "flag"
  "fmt"
  "html/template"
  "io"
  "os"
  "path/filepath"
  "strings"
  "sync"
)

var (
  debug        bool
  outFile      string
  dependencies chan Dependency
  wg           sync.WaitGroup
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

  <dependencies>
{{range .}}
    <dependency>
      <groupId>{{.GroupId}}</groupId>
      <artifactId>{{.ArtifactId}}</artifactId>
      <version>{{.Version}}</version>
    </dependency>
{{end}}
  </dependencies>
</project>`

type Dependency struct {
  GroupId    string `json:"groupId"`
  ArtifactId string `json:"artifactId"`
  Version    string `json:"version"`
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

// function matching the filepath.WalkFunc
func walkJar(path string, info os.FileInfo, err error) error {
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
      debugf("%s <- %s", hash, filepath.Base(path))
      wg.Add(1)
      if err = jboss(hash, dependencies); err != nil {
        return err
      }
    } else {
      return filepath.SkipDir
    }

  }
  return nil
}

func collect(dependencies <-chan Dependency) {
  pomTmpl := template.Must(template.New("pom").Parse(pomTemplate))
  var deps []Dependency
  for {
    dep, more := <-dependencies
    fmt.Println("collect", dep, more)
    if more {
      deps = append(deps, dep)
    } else {
      fmt.Println(deps)
      buf := new(bytes.Buffer)
      err := pomTmpl.Execute(buf, deps)
      if err == nil {
        fmt.Println(buf.String())
      }
      wg.Done()
      return
    }
  }
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
  dependencies = make(chan Dependency, 1)
  wg.Add(2)
  go func() {
    for i := 0; i < nDirs; i++ {
      filepath.Walk(flag.Arg(i), walkJar)
    }
    close(dependencies)
    wg.Done()
  }()
  go collect(dependencies)
  wg.Wait()
}
