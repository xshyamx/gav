package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

//{"results":[{"repoKey":"jcenter-cache","name":"poi-ooxml-3.10.1.jar","modifiedDate":1408341120000,"modifiedString":"18-08-14 05:52:00 +00:00","actions":["Download","ShowInTree"],"downloadLink":"https://oss.jfrog.org/artifactory/jcenter-cache/org/apache/poi/poi-ooxml/3.10.1/poi-ooxml-3.10.1.jar","relativePath":"org/apache/poi/poi-ooxml/3.10.1/poi-ooxml-3.10.1.jar","relativeDirPath":"org/apache/poi/poi-ooxml/3.10.1"}],"searchExpression":"0c62b1db67f2a7cafd4dd55c41256a2fa0793191","message":"Search Results - 1 Items"}

type ArtifactoryResponse struct {
	Data []ArtifactoryArtifact `json:"results"`
}
type ArtifactoryArtifact struct {
	Name string `json:"name"`
	Path string `json:"relativeDirPath"`
}

type ArtifactoryPayload struct {
	Checksum string `json:"checksum"`
	Search   string `json:"search"`
}

func getPayload(sha1 string) ArtifactoryPayload {
	return ArtifactoryPayload{
		Checksum: sha1,
		Search:   "checksum",
	}
}

func artifactoryFunc(repoName, reqUrl string) func(string) (Dependency, error) {
	return func(sha1 string) (dep Dependency, err error) {
		var (
			content      []byte
			req          *http.Request
			res          *http.Response
			repoResponse ArtifactoryResponse
		)
		payload := getPayload(sha1)
		buf, err := json.Marshal(payload)
		if err != nil {
			return
		}
		req, err = http.NewRequest(http.MethodPost, reqUrl, bytes.NewReader(buf))
		if err != nil {
			return
		}
		req.Header.Add("Accept", "application/json")
		req.Header.Add("Content-Type", "application/json")
		res, err = http.DefaultClient.Do(req)
		if err != nil {
			return
		}
		if res.StatusCode != http.StatusOK {
			err = fmt.Errorf("Expected %d got %d for %s", http.StatusOK, res.StatusCode, reqUrl)
			return
		}
		defer res.Body.Close()
		content, err = ioutil.ReadAll(res.Body)
		if err != nil {
			err = fmt.Errorf("Failed to read response for %s", reqUrl)
			return
		}
		//debugf("%s -> %s", repoName, string(content))
		err = json.Unmarshal(content, &repoResponse)
		if err != nil {
			err = fmt.Errorf("Failed to parse JSON for %s", reqUrl)
			return
		}
		debugf("%s Response %v", repoName, repoResponse)
		for _, aft := range repoResponse.Data {
			if strings.HasSuffix(aft.Name, ".jar") {
				splits := strings.Split(aft.Path, "/")
				l := len(splits)
				dep = Dependency{
					GroupId:    strings.Join(splits[:l-2], "."),
					ArtifactId: splits[l-2],
					Version:    splits[l-1],
				}
				return
			}
		}
		err = fmt.Errorf("Failed to find matching dependency")
		return
	}
}
