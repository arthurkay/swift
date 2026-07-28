BINARY           := swift
CMD_DIR          := ./cmd/swift
INSTALL_DIR      := /usr/local/bin
SWIFT_HOME       := /var/swift
GOLANG_CROSS_VERSION ?= v1.17.6
PACKAGE_NAME     := github.com/arthurkay/swift

SYSROOT_DIR      ?= sysroots
SYSROOT_ARCHIVE  ?= sysroots.tar.bz2

# ---------- detect distro ----------
DISTRO_FAMILY    := $(shell ./scripts/detect-distro.sh)

# ---------- phony ----------
.PHONY: help deps deps-deb deps-rpm build install uninstall clean \
        setup lint test release-dry-run release \
        sysroot-pack sysroot-unpack

# ---------- default ----------
help: ## Show this help
	@grep -E '^[a-zA-Z_/-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

# ====================== system dependencies ======================

deps: ## Install all system prerequisites
ifeq ($(DISTRO_FAMILY),deb)
	$(MAKE) deps-deb
else ifeq ($(DISTRO_FAMILY),rpm)
	$(MAKE) deps-rpm
else
	$(error Unsupported distro. Install manually: libvirt-dev, libvirt-daemon-system, qemu-utils, qemu-kvm, genisoimage, cloud-init)
endif
	@echo "\033[32mAll system dependencies installed.\033[0m"

deps-deb: ## Install dependencies on Debian / Ubuntu
	@echo "\033[33mInstalling system dependencies (apt)...\033[0m"
	sudo apt-get update
	sudo apt-get install -y \
		libvirt-dev \
		libvirt-daemon-system \
		pkg-config \
		qemu-utils \
		qemu-kvm \
		genisoimage \
		cloud-init \
		golang-go
	@echo "\033[33mAdding $(USER) to libvirt group...\033[0m"
	sudo usermod -aG libvirt $(USER) || true
	@echo "\033[32mDone. You may need to log out and back in for group changes to take effect.\033[0m"

deps-rpm: ## Install dependencies on RHEL / Fedora / CentOS / openSUSE
	@echo "\033[33mInstalling system dependencies (dnf/yum)...\033[0m"
	sudo dnf install -y \
		libvirt-devel \
		libvirt-daemon \
		pkgconf-pkg-config \
		qemu-img \
		qemu-kvm \
		mkisofs \
		cloud-init \
		golang || \
	sudo yum install -y \
		libvirt-devel \
		libvirt-daemon \
		pkgconfig \
		qemu-img \
		qemu-kvm \
		mkisofs \
		cloud-init \
		golang
	@echo "\033[33mAdding $(USER) to libvirt group...\033[0m"
	sudo usermod -aG libvirt $(USER) || true
	@echo "\033[32mDone. You may need to log out and back in for group changes to take effect.\033[0m"

# ====================== setup ======================

setup: deps ## Install deps, create /var/swift, start and enable libvirtd
	@echo "\033[33mCreating swift home directory $(SWIFT_HOME)...\033[0m"
	sudo mkdir -p $(SWIFT_HOME)
	sudo chmod 0775 $(SWIFT_HOME)
	sudo chown $(USER):$(USER) $(SWIFT_HOME)
	@echo "\033[33mEnabling and starting libvirtd...\033[0m"
	sudo systemctl enable --now libvirtd
	@echo "\033[32mSwift home ready at $(SWIFT_HOME). libvirtd enabled and running.\033[0m"

# ====================== build / install ======================

build: ## Build the swift binary
	CGO_ENABLED=1 go build -o $(BINARY) $(CMD_DIR)
	@echo "\033[32mBuilt ./$(BINARY)\033[0m"

install: build ## Build and install to $(INSTALL_DIR)
	sudo install -m 755 $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "\033[32mInstalled $(BINARY) to $(INSTALL_DIR)\033[0m"

uninstall: ## Remove swift from $(INSTALL_DIR)
	sudo rm -f $(INSTALL_DIR)/$(BINARY)
	@echo "\033[32mUninstalled $(BINARY) from $(INSTALL_DIR)\033[0m"

clean: ## Remove build artifacts
	rm -f $(BINARY)
	rm -rf dist/

# ====================== lint / test ======================

lint: ## Run go vet
	go vet ./...

test: ## Run tests
	go test ./...

# ====================== release (goreleaser) ======================

release-dry-run: ## Dry-run release via Docker
	@docker run \
		--rm \
		--privileged \
		-e CGO_ENABLED=1 \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME) \
		-v $(CURDIR)/sysroot:/sysroot \
		-w /go/src/$(PACKAGE_NAME) \
		goreleaser/goreleaser-cross:$(GOLANG_CROSS_VERSION) \
		--rm-dist --skip-validate --skip-publish

release: ## Run a full release via Docker
	@if [ ! -f ".release-env" ]; then \
		echo "\033[91m.release-env is required for release\033[0m";\
		exit 1;\
	fi
	docker run \
		--rm \
		--privileged \
		-e CGO_ENABLED=1 \
		--env-file .release-env \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME) \
		-v $(CURDIR)/sysroot:/sysroot \
		-w /go/src/$(PACKAGE_NAME) \
		goreleaser/goreleaser-cross:$(GOLANG_CROSS_VERSION) \
		release --rm-dist

# ====================== sysroot ======================

sysroot-pack: ## Pack sysroots for cross-compilation
	@tar cf - $(SYSROOT_DIR) -P | pv -s $[$(du -sk $(SYSROOT_DIR) | awk '{print $$1}') * 1024] | pbzip2 > $(SYSROOT_ARCHIVE)

sysroot-unpack: ## Unpack sysroots for cross-compilation
	@pv $(SYSROOT_ARCHIVE) | pbzip2 -cd | tar -xf -
