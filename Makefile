.PHONY: help build test clean install test-iso test-iso-headless lint fmt deps iso bootstrap minimal-rootfs minimal-iso test-minimal-iso-headless docs

BINARY_NAME=agnostic

# Diretório base — TUDO relacionado ao AgnostikOS fica aqui.
# Nunca altere para apontar para fora deste caminho.
AGNOSTICOS_BASE ?= /mnt/data/agnostikOS

# Diretório de build do binário Go (dentro do base)
BUILD_DIR=$(AGNOSTICOS_BASE)/build

# Diretório do RootFS / LFS (dentro do base)
LFS ?= $(AGNOSTICOS_BASE)/rootfs

# Variável de ambiente usada pelo binário em runtime para resolver o rootfs
export AGNOSTICOS_ROOT=$(LFS)

# Flags extras repassadas ao binário. Exemplos:
#   make bootstrap ARGS="--skip-grub"
#   make bootstrap ARGS="--skip-kernel --skip-busybox --skip-initramfs --skip-grub"
#   make bootstrap ARGS="--force"
ARGS ?=

GO=go
LDFLAGS=-ldflags "-X github.com/ElioNeto/agnostikos/cmd/agnostic.Version=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev) -X github.com/ElioNeto/agnostikos/cmd/agnostic.Commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)"

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "  Use ARGS=\"...\" to pass extra flags to bootstrap/iso:"
	@echo "    make bootstrap ARGS=\"--skip-grub\""
	@echo "    make bootstrap ARGS=\"--skip-kernel --skip-busybox --skip-initramfs --skip-grub\""
	@echo "    make bootstrap ARGS=\"--force\""
	@echo "    make iso       ARGS=\"--uefi\""

# Garante que o diretório base exista antes de qualquer build
# (CI runners podem não ter /mnt/data; usa sudo com fallback e chown)
$(AGNOSTICOS_BASE):
	@if ! mkdir -p $@ 2>/dev/null; then \
		sudo mkdir -p $@; \
		sudo chown $$(id -u):$$(id -g) $@; \
	fi

build: $(AGNOSTICOS_BASE) ## Build the CLI binary
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

install: build ## Install binary to /usr/local/bin
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)

test: ## Run unit tests
	$(GO) test -v -race -coverprofile=$(AGNOSTICOS_BASE)/coverage.out ./...

test-iso: ## Test ISO in QEMU
	@bash scripts/run-qemu.sh $(BUILD_DIR)/agnostikos-latest.iso

# Timeout for headless ISO test in CI (300s for TCG emulation)
TEST_ISO_TIMEOUT ?= 300

test-iso-headless: ## Test ISO in QEMU (headless, for CI)
	HEADLESS=1 BOOT_TIMEOUT=$(TEST_ISO_TIMEOUT) bash scripts/run-qemu.sh $(BUILD_DIR)/agnostikos-latest.iso

# Minimal RootFS for CI testing (no toolchain, uses host kernel)
# Diretório do rootfs mínimo (para testes sem toolchain completa)
MINIMAL_ROOTFS_DIR=$(AGNOSTICOS_BASE)/minimal-rootfs

minimal-rootfs: ## Prepare minimal rootfs with host kernel (no toolchain)
	@bash scripts/prepare-minimal-rootfs.sh

minimal-iso: build minimal-rootfs ## Build test ISO from minimal rootfs (host kernel + test initramfs)
	@$(BUILD_DIR)/$(BINARY_NAME) iso \
		--rootfs $(MINIMAL_ROOTFS_DIR) \
		--output $(BUILD_DIR)/agnostikos-latest.iso \
		--test \
		$(ARGS)

test-minimal-iso-headless: minimal-iso ## Build minimal ISO and test it headless in QEMU (for CI)
	HEADLESS=1 BOOT_TIMEOUT=$(TEST_ISO_TIMEOUT) bash scripts/run-qemu.sh $(BUILD_DIR)/agnostikos-latest.iso

test-boot-integration: build ## Run full boot integration test (bootstrap → ISO → QEMU)
	@bash scripts/test-boot-integration.sh --timeout $(TEST_ISO_TIMEOUT)

test-boot-integration-uefi: build ## Run full boot integration test (UEFI mode)
	@bash scripts/test-boot-integration.sh --uefi --timeout $(TEST_ISO_TIMEOUT)

lint: ## Run golangci-lint
	golangci-lint run ./...

fmt: ## Format Go code
	$(GO) fmt ./...

clean: ## Clean build artifacts (remove build/ e tmp/ dentro do base dir)
	@rm -rf $(BUILD_DIR) $(AGNOSTICOS_BASE)/tmp $(AGNOSTICOS_BASE)/coverage.out

deps: ## Download Go dependencies
	$(GO) mod download
	$(GO) mod tidy

iso: build ## Build ISO from RootFS — output vai para $(BUILD_DIR)/agnostikos-latest.iso
	@$(BUILD_DIR)/$(BINARY_NAME) iso \
		--rootfs $(LFS) \
		--output $(BUILD_DIR)/agnostikos-latest.iso \
		$(ARGS)

bootstrap: build ## (internal) Bootstrap RootFS into $(LFS) — use 'build' instead
	@sudo $(BUILD_DIR)/$(BINARY_NAME) bootstrap $(ARGS)

docs: ## Generate man pages and Markdown docs
	$(GO) run ./cmd/agnostic/gen-docs/

dev: ## Run in development mode
	@$(GO) run . --help

package: build ## Build .deb and .rpm packages
	@bash scripts/package.sh

.PHONY: package
.DEFAULT_GOAL := help
