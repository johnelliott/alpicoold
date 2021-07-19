# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=alpicoold
BINARY_UNIX=$(BINARY_NAME)_unix
BINARY_RASPI=$(BINARY_NAME)_raspi
BINARY_WINDOWS=$(BINARY_NAME)_windows

# TODO fix this and make it come from ansible inventory
GOARM=6

all: test build
	build:
	$(GOBUILD) -o $(BINARY_NAME) -v
test:
	$(GOTEST) -v ./...
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f $(BINARY_RASPI)
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
				-o $(BINARY_RASPI)

#build-linux:
#       CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v
#build-windows:
#        CGO_ENABLED=1 GOOS=windows GOARCH=386 $(GOBUILD) -o $(BINARY_WINDOWS) -v
#docker-build:
#        docker run --rm -it -v "$(GOPATH)":/go -w /go/src/bitbucket.org/rsohlich/makepost golang:latest go build -o "$(BINARY_UNIX)" -v

# ansible things
deploy: clean build
	ansible-playbook -i ~/inventory.yml deploy.yml

# local stuff replaced by ansible
journal:
	journalctl -u alpicoold -f
	#journalctl -u alpicoold
	#journalctl -u alpicoold.service -f -o json |jq -c -f /home/pi/bluetooth-fridge/journal.jq
status:
	sudo systemctl status alpicoold

daemon-reload:
	sudo systemctl daemon-reload
	start: daemon-reload
	sudo systemctl start alpicoold
stop:
	sudo systemctl stop alpicoold

restart: daemon-reload
	sudo systemctl restart alpicoold
	enable: daemon-reload
	sudo systemctl enable alpicoold
disable:
	sudo systemctl disable alpicoold

update: build restart
