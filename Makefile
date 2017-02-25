
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

release: build
	export GITHUB_API=https://github.ibm.com/api/v3
	ghr -t dc4acf704f245481569f42c3651b81c478038045 ${VERSION} dist/

test:
	go vet ./src/...
	go test -v -race -cover ./src/...

clean:
	rm -frv dist