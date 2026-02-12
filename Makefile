define generate_mocks
    cd src && mockery --exported --packageprefix mocked --name=$(3) --recursive --case underscore --dir ./$(1) --output ./internal/test/mocks/$(2)
endef

SHELL := /bin/bash

VERSION := 2.6.2
APP_NAME := easee

ARCH ?= armhf

OUT_DIR := package/build
DEB_DIR := package/debian
LOG_DIR := /var/log/thingsplex/$(APP_NAME)
CONTROL_DIR := $(DEB_DIR)/DEBIAN
TARGET_PKG := $(OUT_DIR)/$(APP_NAME)_$(VERSION)_$(ARCH).deb
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

build-local:
	cd src ; go build -ldflags="-s -w -X main.Version=$(VERSION)" -o ../$(APP_NAME) main.go

build-arm:
	cd src ; GOOS=linux GOARCH=arm GOARM=6 go build -ldflags="-s -w -X main.Version=$(VERSION)" -o ../$(TARGET_BIN) main.go

build-linux-amd64:
	cd src ; GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.Version=$(VERSION)" -o ../$(TARGET_BIN) main.go

build-mac-amd64:
	cd src ; GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.Version=$(VERSION)" -o ../$(TARGET_BIN) main.go

build-win-amd64:
	cd src ; GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.Version=$(VERSION)" -o ../$(APP_NAME).exe main.go

configure:
	mkdir -p $(CONTROL_DIR)
	mkdir -p $(BIN_DIR)
	mkdir -p $(LOG_DIR)
	printf '%s\n' \
	  "Package: $(APP_NAME)" \
	  "Version: $(VERSION)" \
	  "Section: non-free/misc" \
	  "Priority: optional" \
	  "Architecture: $(ARCH)" \
	  "Maintainer: Futurehome AS <dev@futurehome.no>" \
	  "Description: Futurehome Easee EV charger adapter" \
	  > $(CONTROL_DIR)/control

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
	ssh -t -p $(PORT) $(REMOTE_HOST) "su - fh -c 'sudo dpkg -i /home/fhtunnel/$(APP_NAME)_$(VERSION)_$(ARCH).deb'"

test:
	rm -f test_coverage.out || true

	@echo "Run tests"
	cd src && go test -p 1 -count 1 -failfast -covermode=atomic -coverprofile=profile_full.cov -coverpkg=./... ./...

	@echo "Preparecoverage report"
	cd src && cat profile_full.cov | grep -v .pb.go | grep -v mock | grep -v test > test_coverage.out
	mv src/test_coverage.out .
	rm -f src/profile_full.cov

generate-mocks:
	find ./src/internal/test/mocks -type f -not -name "*_helper.go" | xargs rm -rf
	$(call generate_mocks,"internal/api","api","Authenticator|Client|NewAPIClient")
	$(call generate_mocks,"internal/app","app","Application")
	$(call generate_mocks,"internal/cache","cache","Cache")
	$(call generate_mocks,"internal/db","db","ChargingSessionStorage")
	$(call generate_mocks,"internal/signalr","signalr","Client")

                        

.PHONY: all clean test generate-mocks configure package-deb deb-arm deb-amd upload deploy
