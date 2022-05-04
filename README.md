A [Distribution](https://github.com/distribution/distribution) based registry
that returns 429: Too many requests based on user configuration.

## Usage

1. run the registry: `go run main.go`
1. configure the quota:
    * via terminal: `curl -XPOST -d 'c=1' localhost:8080` (replace 1 with the desired quota)
    * via browser: `open http://localhost:8080`
1. check the quota: `open http://localhost:8080`
1. use the registry (i.e `podman pull localhost:8080/ns/image:tag`)
