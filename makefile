# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=alpicoold
#BINARY_UNIX=$(BINARY_NAME)_unix
BINARY_RASPI=$(BINARY_NAME)_raspi_arm
#BINARY_WINDOWS=$(BINARY_NAME)_windows

all: test build
	build:
	$(GOBUILD) -o $(BINARY_NAME) -v
test:
	$(GOTEST) -v ./...
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	#rm -f $(BINARY_UNIX)
	rm -f $(BINARY_RASPI)$(GOARM)
run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./...
	./$(BINARY_NAME)
deps:
	echo TODO add go get dependencies
	#$(GOGET) github.com/markbates/goth
	#$(GOGET) github.com/markbates/pop


build: build-raspi

# Cross compilation from macos to linux, needs homebrew install of CC tools
build-raspi:
	CGO_ENABLED=1 GOOS=linux GOARCH=arm GOARM=$(GOARM) CC=arm-linux-musleabihf-gcc \
				$(GOBUILD) -v \
				--ldflags '-linkmode external -extldflags "-static"' \
				-o $(BINARY_RASPI)$(GOARM)
#build-linux:
#       CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v
#build-windows:
#        CGO_ENABLED=1 GOOS=windows GOARCH=386 $(GOBUILD) -o $(BINARY_WINDOWS) -v
#docker-build:
#        docker run --rm -it -v "$(GOPATH)":/go -w /go/src/bitbucket.org/rsohlich/makepost golang:latest go build -o "$(BINARY_UNIX)" -v
