TARGET_PATH = bin
GOARCH = GOARCH=amd64
VERSION = 1.0.0

buildWindows:
	env GOOS=windows $(GOARCH) go build -o ./bin/aps-Windows-$(VERSION).exe

buildMacOS:
	env GOOS=darwin $(GOARCH) go build -o ./bin/aps-MacOS-$(VERSION)

buildLinux:
	env GOOS=linux $(GOARCH) go build -o ./bin/aps-Linux-$(VERSION)

build: buildWindows buildMacOS buildLinux

clean:
	rm -rf bin

all: clean build