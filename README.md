# gav #

gav is a utility to generate a pom.xml file from a given set of jar files. It can be used to migrate legacy ant based projects which need to store its dependencies in a lib folder. `gav` stands for **groupId** **artifactId** and **version** which constitute a dependency entry in the pom.xml.


## Build & Run ##

### Build ###

``` sh
go build
```

### Run ###

```sh
go run *.go

# or 

./gav
```
