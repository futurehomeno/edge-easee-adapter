SHELL := /bin/bash

TAG := 2.6.0
APP_NAME := easee

ARCH ?= armhf

OUT_DIR := package/build
DEB_DIR := package/debian
TARGET_PKG := $(OUT_DIR)/$(APP_NAME)_$(TAG)_$(ARCH).deb
BIN_DIR := $(DEB_DIR)/opt/thingsplex/$(APP_NAME)
TARGET_BIN := $(BIN_DIR)/$(APP_NAME)

REMOTE_HOST := fhtunnel@3.255.43.28
PORT := 8000

all: deb-arm

clean:
	-rm -f $(OUT_DIR)/*
	-rm -f $(TARGET_BIN)
	-rm -f $(APP_NAME)
	-rm -f $(APP_NAME).exe
	mkdir -p $(BIN_DIR)

build-local:
	cd src ; go build -ldflags="-s -w -X main.Version=$(TAG)" -o ../$(APP_NAME) main.go

build-arm:
	cd src ; GOOS=linux GOARCH=arm GOARM=6 go build -ldflags="-s -w -X main.Version=$(TAG)" -o ../$(TARGET_BIN) main.go

build-linux-amd64:
	cd src ; GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.Version=$(TAG)" -o ../$(TARGET_BIN) main.go

build-mac-amd64:
	cd src ; GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.Version=$(TAG)" -o ../$(TARGET_BIN) main.go

build-win-amd64:
	cd src ; GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.Version=$(TAG)" -o ../$(APP_NAME).exe main.go

configure:
	python3 ./scripts/config_env.py $(DEB_DIR)/DEBIAN $(TAG) $(ARCH)

package-deb:
	chmod 755 $(DEB_DIR)
	chmod 644 $(DEB_DIR)/DEBIAN/control
	chmod -R g-w $(DEB_DIR)

	@if command -v dpkg-deb >/dev/null; then \
		echo "Using local dpkg-deb"; \
		fakeroot dpkg-deb -Zxz -b $(DEB_DIR) $(TARGET_PKG); \
	else \
		echo "Using docker dpkg-deb"; \
		docker run --rm -v "$$(pwd)":/build -w /build debian:stable-slim \
			bash -c "\
			    apt-get update >/dev/null && \
			    apt-get install -y --no-install-recommends dpkg-dev >/dev/null && \
			    fakeroot dpkg-deb -Zxz -b $(DEB_DIR) $(TARGET_PKG)"; \
	fi

	@echo "Debian package created → $(TARGET_PKG)"

deb-arm: ARCH=armhf
deb-arm: clean configure build-arm package-deb

deb-amd: ARCH=amd64
deb-amd: clean configure build-linux-amd64 package-deb

upload:
	@echo "Uploading..."
	rsync -avz -e "ssh -p $(PORT)" $(TARGET_PKG) $(REMOTE_HOST):/home/fhtunnel/

deploy: upload
	ssh -t -p $(PORT) $(REMOTE_HOST) "su - fh -c 'sudo dpkg -i /home/fhtunnel/$(APP_NAME)_$(TAG)_$(ARCH).deb'"

test:
	rm -f test_coverage.out || true

	@echo "Running tests"
	cd src && go test -p 1 -count 1 -v -failfast -covermode=atomic -coverprofile=profile_full.cov -coverpkg=./... ./...

	@echo "Preparing coverage report"
	cd src && cat profile_full.cov | grep -v .pb.go | grep -v mock | grep -v test > test_coverage.out
	mv src/test_coverage.out .
	rm -f src/profile_full.cov

mocks:
	cd ./src && mockery --dir ./internal --all --output ./internal/test/mocks --disable-version-string


.PHONY: all clean test mocks configure package-deb deb-arm deb-amd upload deploy
