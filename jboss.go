package main

import (
  "encoding/json"
  "fmt"
  "io/ioutil"
  "net/http"
)

//{"totalCount":0,"from":-1,"count":-1,"tooManyResults":false,"collapsed":false,"repoDetails":[],"data":[]}

type JbossResponse struct {
  Count int          `json:"totalCount"`
  Data  []Dependency `json:"data"`
}

func jboss(sha1 string) (dep Dependency, err error) {
  var (
    content      []byte
    req          *http.Request
    res          *http.Response
    repoResponse JbossResponse
  )
  reqUrl := fmt.Sprintf("https://repository.jboss.org/nexus/service/local/lucene/search?sha1=%s", sha1)
  req, err = http.NewRequest(http.MethodGet, reqUrl, nil)
  if err != nil {
    return
  }
  req.Header.Add("Accept", "application/json")
  res, err = http.DefaultClient.Do(req)
  if err != nil {
    return
  }
  if res.StatusCode != http.StatusOK {
    err = fmt.Errorf("Expected %d got %s for %s", http.StatusOK, res.StatusCode, reqUrl)
    return
  }
  defer res.Body.Close()
  content, err = ioutil.ReadAll(res.Body)
  if err != nil {
    err = fmt.Errorf("Failed to read response for %s", reqUrl)
    return
  }
  //debugf("%s", string(content))
  err = json.Unmarshal(content, &repoResponse)
  if err != nil {
    err = fmt.Errorf("Failed to parse JSON for %s", reqUrl)
    return
  }
  if repoResponse.Count > 0 && len(repoResponse.Data) > 0 {
    dep = repoResponse.Data[0]
    debugf("From jboss : %v", dep)
    return
  } else {
    err = fmt.Errorf("Failed to find matching dependency")
  }
  return
}
func jbossAsync(sha1 string, dependencies chan<- Dependency) error {
  dep, err := jboss(sha1)
  if err != nil {
    return err
  }
  dependencies <- dep
  return nil
}
