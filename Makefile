# Variables
VERSION ?= 0.1.0

# .env
ifneq (,$(wildcard ./.env))
	include .env
	export
endif

## Print this message and exit
.PHONY: help
help:
	@cat $(MAKEFILE_LIST) | awk '														\
		/^([0-9a-z-]+):.*$$/ {															\
			if (description[0] != "") {													\
				printf("\x1b[36mmake %s\x1b[0m\n", substr($$1, 0, length($$1)-1));		\
				for (i in description) {												\
					printf("| %s\n", description[i]);									\
				}																		\
				printf("\n");															\
				split("", description);													\
				descriptionIndex = 0;													\
			}																			\
		}																				\
		/^##/ {																			\
			description[descriptionIndex++] = substr($$0, 4);							\
		}																				\
	'

## Build development version
.PHONY: dev
dev:
	go build

## Build production version
.PHONY: prod
prod:
	CGO_ENABLED=0 go build -ldflags='-s -w -X main.version=$(VERSION)'

## Install to ${GOPATH}/bin
.PHONY: install
install:
	CGO_ENABLED=0 go install -ldflags='-s -w -X main.version=$(VERSION)'

## Build and create a release
.PHONY: release
release:
	if git rev-parse v$(VERSION) >/dev/null 2>&1; then echo "ERROR: Tag 'v$(VERSION)' already exists"; exit 1; fi
	rm -rf ./releases/v$(VERSION)
	mkdir -p ./releases/v$(VERSION)
	GOOS=linux   GOARCH=amd64   CGO_ENABLED=0 go build -o ./releases/v$(VERSION)/nwatch-v$(VERSION)-linux-amd64        -ldflags='-s -w -X main.version=$(VERSION)'
	GOOS=linux   GOARCH=386     CGO_ENABLED=0 go build -o ./releases/v$(VERSION)/nwatch-v$(VERSION)-linux-386          -ldflags='-s -w -X main.version=$(VERSION)'
	GOOS=linux   GOARCH=arm64   CGO_ENABLED=0 go build -o ./releases/v$(VERSION)/nwatch-v$(VERSION)-linux-arm64        -ldflags='-s -w -X main.version=$(VERSION)'
	GOOS=linux   GOARCH=arm     CGO_ENABLED=0 go build -o ./releases/v$(VERSION)/nwatch-v$(VERSION)-linux-arm          -ldflags='-s -w -X main.version=$(VERSION)'
	GOOS=linux   GOARCH=ppc64le CGO_ENABLED=0 go build -o ./releases/v$(VERSION)/nwatch-v$(VERSION)-linux-ppc64le      -ldflags='-s -w -X main.version=$(VERSION)'
	GOOS=linux   GOARCH=s390x   CGO_ENABLED=0 go build -o ./releases/v$(VERSION)/nwatch-v$(VERSION)-linux-s390x        -ldflags='-s -w -X main.version=$(VERSION)'
	GOOS=darwin  GOARCH=amd64   CGO_ENABLED=0 go build -o ./releases/v$(VERSION)/nwatch-v$(VERSION)-darwin-amd64       -ldflags='-s -w -X main.version=$(VERSION)'
	GOOS=darwin  GOARCH=arm64   CGO_ENABLED=0 go build -o ./releases/v$(VERSION)/nwatch-v$(VERSION)-darwin-arm64       -ldflags='-s -w -X main.version=$(VERSION)'
	GOOS=windows GOARCH=amd64   CGO_ENABLED=0 go build -o ./releases/v$(VERSION)/nwatch-v$(VERSION)-windows-amd64.exe  -ldflags='-s -w -X main.version=$(VERSION)'
	gh release create v0.1.0 --title='' --notes='' ./releases/v0.1.0/nwatch-*
