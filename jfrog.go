package main

var jfrog = artifactoryFunc("JFrog", "https://oss.jfrog.org/ui/artifactsearch/checksum")

func jfrogAsync(sha1 string, dependencies chan<- Dependency) error {
	dep, err := jfrog(sha1)
	if err != nil {
		return err
	}
	dependencies <- dep
	return nil
}
