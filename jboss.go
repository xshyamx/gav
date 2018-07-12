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

func jboss(sha1 string, dependencies chan<- Dependency) error {
  defer wg.Done()
  var (
    err          error
    content      []byte
    req          *http.Request
    res          *http.Response
    repoResponse JbossResponse
  )
  reqUrl := fmt.Sprintf("https://repository.jboss.org/nexus/service/local/lucene/search?sha1=%s", sha1)
  req, err = http.NewRequest(http.MethodGet, reqUrl, nil)
  if err != nil {
    return nil
  }
  req.Header.Add("Accept", "application/json")
  res, err = http.DefaultClient.Do(req)
  if err != nil {
    return err
  }
  if res.StatusCode != http.StatusOK {
    return fmt.Errorf("Expected %d got %s for %s", http.StatusOK, res.StatusCode, reqUrl)
  }
  defer res.Body.Close()
  content, err = ioutil.ReadAll(res.Body)
  if err != nil {
    return fmt.Errorf("Failed to read response for %s", reqUrl)
  }
  //debugf("%s", string(content))
  err = json.Unmarshal(content, &repoResponse)
  if err != nil {
    return fmt.Errorf("Failed to parse JSON for %s", reqUrl)
  }
  if repoResponse.Count > 0 && len(repoResponse.Data) > 0 {
    debugf("%v", repoResponse.Data[0])
    dependencies <- repoResponse.Data[0]
    //    os.Exit(1)
  }
  return nil
}
