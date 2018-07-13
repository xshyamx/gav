package main

import (
  "encoding/json"
  "fmt"
  "io/ioutil"
  "net/http"
)

//{"responseHeader":{"status":0,"QTime":1,"params":{"q":"1:\"cf993e250ff71804754ec2734a16f23c0be99f70\"","indent":"off","fl":"id,g,a,v,p,ec,timestamp,tags","sort":"score desc,timestamp desc,g asc,a asc,v desc","rows":"20","wt":"json","version":"2.2"}},"response":{"numFound":1,"start":0,"docs":[{"id":"commons-codec:commons-codec:1.5","g":"commons-codec","a":"commons-codec","v":"1.5","p":"jar","timestamp":1301016846000,"ec":["-site.xml","-javadoc.jar","-sources.jar",".jar",".pom"],"tags":["encoders","decoders","phonetic","such","simple","package","codec","contains","collection","used","addition","formats","base64","hexadecimal","widely","utilities","maintains","encoding","various","also","these","encoder"]}]}}

type CentralWrapper struct {
  Response CentralResponse `json:"response"`
}
type CentralResponse struct {
  Count int                 `json:"numFound"`
  Data  []CentralDependency `json:"docs"`
}
type CentralDependency struct {
  GroupId    string `json:"g"`
  ArtifactId string `json:"a"`
  Version    string `json:"v"`
  Packaging  string `json:"p"`
}

func central(sha1 string) (dep Dependency, err error) {
  var (
    content      []byte
    req          *http.Request
    res          *http.Response
    repoWrapper  CentralWrapper
    repoResponse CentralResponse
  )
  reqUrl := fmt.Sprintf("http://search.maven.org/solrsearch/select?q=1:%%22%s%%22&rows=20&wt=json", sha1)
  //debugf("requesting %s", reqUrl)
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
  err = json.Unmarshal(content, &repoWrapper)
  if err != nil {
    err = fmt.Errorf("Failed to parse JSON for %s", reqUrl)
    return
  }
  repoResponse = repoWrapper.Response
  //debugf("Central Response %v", repoResponse)
  if repoResponse.Count > 0 {
    for _, cdep := range repoResponse.Data {
      if cdep.Packaging == "jar" {
        dep = Dependency{
          GroupId:    cdep.GroupId,
          ArtifactId: cdep.ArtifactId,
          Version:    cdep.Version,
        }
        debugf("From central : %v", dep)
        return
      }
    }
  } else {
    err = fmt.Errorf("Failed to find matching dependency")
  }
  return
}
func centralAsync(sha1 string, dependencies chan<- Dependency) error {
  dep, err := central(sha1)
  if err != nil {
    return err
  }
  dependencies <- dep
  return nil
}
