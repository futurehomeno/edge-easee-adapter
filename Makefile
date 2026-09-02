define generate_mocks
    cd src && mockery --exported --packageprefix mocked --name=$(3) --recursive --case underscore --dir ./$(1) --output ./internal/test/mocks/$(2)
endef

# generate_external_mocks generates mocks against an upstream package found in the Go module cache.
# Args: $(1) module path (e.g. github.com/futurehomeno/cliffhanger/storage), $(2) output dir, $(3) iface filter
define generate_external_mocks
    cd src && mockery --exported --packageprefix mocked --name=$(3) --case underscore --dir $$(go list -m -f '{{.Dir}}' $$(echo $(1) | awk -F/ '{print $$1"/"$$2"/"$$3}'))/$$(echo $(1) | awk -F/ '{for(i=4;i<=NF;i++) printf (i==4?"":"/")$$i}') --output ./internal/test/mocks/$(2)
endef

SHELL := /bin/bash

VERSION := 3.0.1
APP_NAME := easee

ARCH ?= armhf

OUT_DIR := package/build
DEB_DIR := package/debian
STAGING := $(OUT_DIR)/root
TARGET_PKG = $(OUT_DIR)/$(APP_NAME)_$(VERSION)_$(ARCH).deb

# clean and configure rm -rf under these paths: refuse to run with OUT_DIR
# empty (the clean glob would expand to /* and STAGING to /root).
ifeq ($(strip $(OUT_DIR)),)
$(error OUT_DIR must not be empty)
endif

# Filesystem layout (tracked files from DEB_DIR staged into STAGING, then
# packaged by dpkg-deb - untracked leftovers in DEB_DIR never ship):
#   /usr/bin/easee                           binary
#   /usr/lib/futurehome/easee/migrate.sh     maintainer-script helper
#   /usr/share/futurehome/easee/defaults/    read-only default config (dpkg-owned)
#   /var/lib/futurehome, /var/log/futurehome root-owned namespace parents
#   /var/lib/futurehome/easee/defaults       shipped symlink to the /usr/share
#                                            defaults (dpkg-owned, see postinst)
# The other easee-owned leaves (data/, data.db, the log dir) are created by
# postinst and by the service itself.
BINARY_DIR := $(STAGING)/usr/bin
CONTROL_DIR := $(STAGING)/DEBIAN
VAR_LIB_DIR := $(STAGING)/var/lib/futurehome
VAR_LOG_DIR := $(STAGING)/var/log/futurehome
TARGET_BIN := $(BINARY_DIR)/$(APP_NAME)

REMOTE_HOST := fhtunnel@3.255.43.28
PORT := 8000

build-local:
	cd src ; go build -ldflags="-s -w -X main.Version=$(VERSION) -X main.PackageName=$(APP_NAME)" -o ../$(APP_NAME) main.go

build-arm:
	cd src ; GOOS=linux GOARCH=arm GOARM=6 go build -ldflags="-s -w -X main.Version=$(VERSION) -X main.PackageName=$(APP_NAME)" -o ../$(TARGET_BIN) main.go

build-linux-amd64:
	cd src ; GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.Version=$(VERSION) -X main.PackageName=$(APP_NAME)" -o ../$(TARGET_BIN) main.go

build-mac-amd64:
	cd src ; GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.Version=$(VERSION) -X main.PackageName=$(APP_NAME)" -o ../$(TARGET_BIN) main.go

clean:
	-rm -rf ./$(OUT_DIR)/*
	-rm -f $(APP_NAME)
	-rm -f test_coverage.out

configure:
	rm -rf ./$(STAGING)
	mkdir -p $(STAGING)
	# Stages tracked files only, so stale artifacts in DEB_DIR never ship.
	# Requires a git checkout: building from an exported source tarball
	# (no .git) fails here.
	git ls-files -z $(DEB_DIR) | tar --null -T - -cf - \
		| tar -xf - --strip-components=2 -C $(STAGING)
	mkdir -p $(BINARY_DIR)
	mkdir -p $(VAR_LIB_DIR)
	mkdir -p $(VAR_LOG_DIR)
	mkdir -p $(CONTROL_DIR)

	# Ship the workdir defaults symlink as package content: dpkg then owns it
	# (removed on remove, restored on reinstall). No easee package ever shipped
	# a real directory at this path, so nothing has to be displaced on upgrade.
	mkdir -p $(VAR_LIB_DIR)/$(APP_NAME)
	ln -sfn /usr/share/futurehome/$(APP_NAME)/defaults $(VAR_LIB_DIR)/$(APP_NAME)/defaults

	printf '%s\n' \
	  "Package: $(APP_NAME)" \
	  "Version: $(VERSION)" \
	  "Section: non-free/misc" \
	  "Priority: optional" \
	  "Architecture: $(ARCH)" \
	  "Depends: fh-drop" \
	  "Maintainer: Futurehome AS <dev@futurehome.no>" \
	  "Description: Futurehome Easee EV charger adapter" \
	  > $(CONTROL_DIR)/control

package-deb:
	@test -x $(TARGET_BIN) || \
		{ echo "error: $(TARGET_BIN) missing; use 'make deb-arm' or 'make deb-amd'" >&2; exit 1; }
	chmod 755 $(STAGING)
	chmod 644 $(CONTROL_DIR)/control
	chmod -R g-w $(STAGING)

	@if command -v dpkg-deb >/dev/null && command -v fakeroot >/dev/null; then \
		echo "Using local dpkg-deb"; \
		fakeroot dpkg-deb -Zxz -b $(STAGING) $(TARGET_PKG); \
	else \
		echo "Using docker dpkg-deb"; \
		docker run --rm -v "$$(pwd)":/build -w /build debian:stable-slim \
			bash -c "\
				apt-get update >/dev/null && \
				apt-get install -y --no-install-recommends dpkg-dev fakeroot >/dev/null && \
				fakeroot dpkg-deb -Zxz -b $(STAGING) $(TARGET_PKG)"; \
	fi

	@echo "Debian package created → $(TARGET_PKG)"

deb-arm: ARCH=armhf
deb-arm: clean configure build-arm package-deb

deb-amd: ARCH=amd64
deb-amd: clean configure build-linux-amd64 package-deb

upload:
	@echo "Uploading..."
	rsync -av --info=progress2 -e "ssh -p $(PORT)" $(TARGET_PKG) $(REMOTE_HOST):/home/fhtunnel/

# Developer-only targets. dpkg -i does not resolve the fh-drop dependency;
# developer hubs are assumed to have it installed already.
deploy: upload
	ssh -t -p $(PORT) $(REMOTE_HOST) "su - fh -c 'sudo dpkg -i /home/fhtunnel/$(APP_NAME)_$(VERSION)_$(ARCH).deb'"

install:
	ssh -t -p $(PORT) $(REMOTE_HOST) "su - fh -c 'sudo dpkg -i /home/fhtunnel/$(APP_NAME)_$(VERSION)_$(ARCH).deb'"

test:
	# it may require to generate mocks first
	rm -f test_coverage.out || true

	@echo "Run tests"
	cd src && go test -p 1 -count 1 -failfast -covermode=atomic -coverprofile=profile_full.cov -coverpkg=./... ./...

	@echo "Prepare coverage report"
	cd src && cat profile_full.cov | grep -v .pb.go | grep -v mock | grep -v test > test_coverage.out
	mv src/test_coverage.out .
	rm -f src/profile_full.cov

generate-mocks:
	mkdir -p ./src/internal/test/mocks
	find ./src/internal/test/mocks -type f -not -name "*_helper.go" -delete 2>/dev/null || true
	$(call generate_mocks,"internal/api","api","Authenticator|Client|HTTPClient")
	$(call generate_mocks,"internal/app","app","Application")
	$(call generate_mocks,"internal/cache","cache","Cache")
	$(call generate_mocks,"internal/db","db","ChargingSessionStorage")
	$(call generate_mocks,"internal/signalr","signalr","Client|Manager")
	$(call generate_external_mocks,"github.com/futurehomeno/cliffhanger/storage","storage","Storage")
	$(call generate_external_mocks,"github.com/futurehomeno/cliffhanger/manifest","manifest","Loader")

.PHONY: clean test generate-mocks configure package-deb deb-arm deb-amd build-mac-amd64 build-linux-amd64 build-arm build-local upload deploy install
