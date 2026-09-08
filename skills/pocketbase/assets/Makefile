SHELL := /bin/sh

PB ?= $(CURDIR)/pocketbase
PB_VERSION ?= latest
PB_DATA_DIR ?= $(CURDIR)/pb_data
PB_HOOKS_DIR ?= $(CURDIR)/pb_hooks
PB_MIGRATIONS_DIR ?= $(CURDIR)/pb_migrations
PB_PUBLIC_DIR ?= $(CURDIR)/pb_public
PB_HTTP ?= 127.0.0.1:8090
SPA_MOUNT_PATH ?= page

.PHONY: dev guard-spa-layout install lint migrate test

install:
	@set -eu; \
	system_name="$$(uname -s | tr '[:upper:]' '[:lower:]')"; \
	machine_name="$$(uname -m)"; \
	case "$$system_name/$$machine_name" in \
		darwin/x86_64|darwin/amd64) asset_platform='darwin_amd64' ;; \
		darwin/arm64|darwin/aarch64) asset_platform='darwin_arm64' ;; \
		linux/x86_64|linux/amd64) asset_platform='linux_amd64' ;; \
		linux/arm64|linux/aarch64) asset_platform='linux_arm64' ;; \
		*) echo "Unsupported platform for PocketBase download: $$system_name/$$machine_name" >&2; exit 1 ;; \
	esac; \
	tmp_dir="$$(mktemp -d "$${TMPDIR:-/tmp}/pocketbase-install.XXXXXX")"; \
	trap 'rm -rf "$$tmp_dir"' EXIT HUP INT TERM; \
	release_url='https://api.github.com/repos/pocketbase/pocketbase/releases/latest'; \
	if [ "$(PB_VERSION)" != 'latest' ]; then \
		normalized_version="$$(printf '%s' '$(PB_VERSION)' | sed 's/^v//')"; \
		release_url="https://api.github.com/repos/pocketbase/pocketbase/releases/tags/v$$normalized_version"; \
	fi; \
	curl --fail --silent --show-error --location \
		--header 'Accept: application/vnd.github+json' \
		--header 'User-Agent: pocketbase-makefile-template' \
		"$$release_url" > "$$tmp_dir/release.json"; \
	version="$$(python3 -c 'import json,sys; print(json.load(sys.stdin)["tag_name"].lstrip("v"))' < "$$tmp_dir/release.json")"; \
	asset_name="pocketbase_$${version}_$${asset_platform}.zip"; \
	archive_url="https://github.com/pocketbase/pocketbase/releases/download/v$${version}/$${asset_name}"; \
	checksums_url="https://github.com/pocketbase/pocketbase/releases/download/v$${version}/checksums.txt"; \
	curl --fail --silent --show-error --location "$${archive_url}" -o "$$tmp_dir/archive.zip"; \
	curl --fail --silent --show-error --location "$${checksums_url}" -o "$$tmp_dir/checksums.txt"; \
	expected_checksum="$$(python3 -c 'import sys; target=sys.argv[1]; rows=(line.strip().split() for line in sys.stdin.read().splitlines()); match=next((parts[0].split(":")[-1].lower() for parts in rows if len(parts) >= 2 and parts[-1].lstrip("*") == target), None); print(match) if match is not None else sys.exit(1)' "$$asset_name" < "$$tmp_dir/checksums.txt")"; \
	actual_checksum="$$(python3 -c 'import hashlib, pathlib, sys; print(hashlib.sha256(pathlib.Path(sys.argv[1]).read_bytes()).hexdigest())' "$$tmp_dir/archive.zip")"; \
	if [ "$$expected_checksum" != "$$actual_checksum" ]; then \
		echo "Checksum mismatch for $$asset_name" >&2; \
		exit 1; \
	fi; \
	mkdir -p "$$(dirname '$(PB)')"; \
	unzip -p "$$tmp_dir/archive.zip" pocketbase > "$(PB)"; \
	chmod +x "$(PB)"; \
	mkdir -p "$(PB_DATA_DIR)" "$(PB_HOOKS_DIR)" "$(PB_MIGRATIONS_DIR)" "$(PB_PUBLIC_DIR)/assets"; \
	echo "Installed PocketBase $$version to $(PB)"; \
		echo "Project directories are ready. /$(SPA_MOUNT_PATH) is reserved for SPA routes; keep compiled assets under $(PB_PUBLIC_DIR)/assets or $(PB_PUBLIC_DIR)/dist."

guard-spa-layout:
	@test ! -d "$(PB_PUBLIC_DIR)/$(SPA_MOUNT_PATH)" || { echo "Forbidden directory: $(PB_PUBLIC_DIR)/$(SPA_MOUNT_PATH). /$(SPA_MOUNT_PATH) is a SPA routing mount, not a real asset directory. Use $(PB_PUBLIC_DIR)/assets or $(PB_PUBLIC_DIR)/dist instead." >&2; exit 1; }

lint:
	@test -x "$(PB)" || { echo "PocketBase binary not found at $(PB). Run 'make install' or override PB." >&2; exit 1; }
	@test -d "$(PB_MIGRATIONS_DIR)" || { echo "Missing migrations directory: $(PB_MIGRATIONS_DIR)" >&2; exit 1; }
	@test -d "$(PB_PUBLIC_DIR)" || { echo "Missing public directory: $(PB_PUBLIC_DIR)" >&2; exit 1; }
	@$(MAKE) guard-spa-layout
	@echo "PocketBase project layout looks valid."

migrate:
	@mkdir -p "$(PB_DATA_DIR)"
	@$(MAKE) guard-spa-layout
	@"$(PB)" migrate up \
		--dev \
		--dir "$(PB_DATA_DIR)" \
		--hooksDir "$(PB_HOOKS_DIR)" \
		--migrationsDir "$(PB_MIGRATIONS_DIR)" \
		--publicDir "$(PB_PUBLIC_DIR)"

dev:
	@mkdir -p "$(PB_DATA_DIR)"
	@$(MAKE) guard-spa-layout
	@exec "$(PB)" serve \
		--dev \
		--http "$(PB_HTTP)" \
		--dir "$(PB_DATA_DIR)" \
		--hooksDir "$(PB_HOOKS_DIR)" \
		--migrationsDir "$(PB_MIGRATIONS_DIR)" \
		--publicDir "$(PB_PUBLIC_DIR)"

test:
	@tmp_dir="$$(mktemp -d "$${TMPDIR:-/tmp}/pocketbase-test.XXXXXX")"; \
	trap 'rm -rf "$$tmp_dir"' EXIT HUP INT TERM; \
	$(MAKE) guard-spa-layout; \
	"$(PB)" migrate up \
		--dev \
		--dir "$$tmp_dir/pb_data" \
		--hooksDir "$(PB_HOOKS_DIR)" \
		--migrationsDir "$(PB_MIGRATIONS_DIR)" \
		--publicDir "$(PB_PUBLIC_DIR)"
