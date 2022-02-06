.PHONY: all
PWD = $(shell pwd)
GO = 1.17
DOCKER = docker run --rm -v $(PWD)/deps:/go/ -v $(PWD):/wd/ -w /wd/ golang:$(GO)

all: build

build:
	$(DOCKER) env GOOS=linux GOARCH=arm GOARM=6 go build

clean:
	$(DOCKER) rm -rf deps teleinfo_exporter

mods:
	$(DOCKER) go mod tidy
