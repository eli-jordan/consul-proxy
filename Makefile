
# The semantic version of the tool
VERSION=0.1.0

# The build number is the git commit id
BUILD=`git rev-parse HEAD`

# Define the linker flags to inject the version/build number
LDFLAGS=-ldflags="-X main.Version=$(VERSION) -X  main.Build=$(BUILD)"

# Define what architecture to cross compile for
OS_ARCH=-osarch="linux/amd64 linux/386 darwin/amd64 darwin/386 windows/amd64 windows/386"

.PHONY: clean test release build deps coverage
.DEFAULT_GOAL: build

# Use gox to cross-compile
build:
	gox $(LDFLAGS) $(OS_ARCH) -output="dist/{{.OS}}_{{.Arch}}/consul-proxy" ./src/...

dev:
	go build -o ./dist/consul-proxy-dev ./src/...

# Install
#   - glide for dependency management
#   - gox for cross compiling
#   - ghr for github releases
deps:
	go get -v github.com/Masterminds/glide
	go get -v github.com/mitchellh/gox
	go get -v github.com/tcnksm/ghr
	glide install

# Zips the build artifacts, and creates a github release
#
# Note:
#    The 'ghr' command is used to create a github release
#    and requires a github API token. In travis this is defined via
#    the GITHUB_TOKEN environment variable, locally it is defined vis
#    the github.token git config
release: build
	zip -r dist/consul-proxy.zip dist/*
	export GITHUB_API=https://github.ibm.com/api/v3/
	ghr -u elijordan ${VERSION} dist/consul-proxy.zip

# Run static analysis + unit tests
test:
	go vet ./src/...
	mkdir ./dist
	go test -v -cover -coverprofile dist/cov.out ./src/...

coverage: test
	go tool cover -html=dist/cov.out -o dist/coverage.html

clean:
	rm -frv dist
