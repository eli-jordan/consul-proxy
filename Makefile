
# The semantic version of the tool
VERSION=0.1.0

# The build number is the git commit id
BUILD=`git rev-parse HEAD`

# Define the linker flags to inject the version/build number
LDFLAGS=-ldflags="-X main.Version=$(VERSION) -X  main.Build=$(BUILD)"

# Define what architecture to cross compile for
OS_ARCH=-osarch="linux/amd64 linux/386 darwin/amd64 darwin/386 windows/amd64 windows/386"

# Install
#   - glide for dependency management
#   - gox for cross compiling
#   - ghr for github releases
deps:
	go get -v github.com/Masterminds/glide
	go get -v github.com/mitchellh/gox
	go get -v github.com/tcnksm/ghr
	glide install

build:
	gox $(LDFLAGS) $(OS_ARCH) -output="dist/{{.OS}}_{{.Arch}}/consul-proxy" ./src/...
	zip -r dist/consul-proxy.zip dist/*

# Note, the 'ghr' command is used to create a github release
# and requires a github API token. In travis this is defined via
# the GITHUB_TOKEN environment variable, locally it is defined vis
# the github.token git config
release: build
	export GITHUB_API=https://github.ibm.com/api/v3/
	ghr -u elijordan ${VERSION} dist/consul-proxy.zip

# Run static analysis + unit tests
test:
	go vet ./src/...
	go test -v -race -cover ./src/...

clean:
	rm -frv dist