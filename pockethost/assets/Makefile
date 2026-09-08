# Legacy compatibility wrapper.
# Prefer npm scripts and the pocketbase-pockethost CLI directly.
SHELL := /bin/sh

CLI ?= npx pocketbase-pockethost

.PHONY: build check deploy dev ftp-deploy sftp-deploy health install lint test workflow-install

install:
	npm install

check lint:
	$(CLI) doctor --strict

test:
	$(CLI) test

build:
	@echo "No build step is required by default. Keep editing pb_public/ directly."

dev:
	$(CLI) dev

deploy:
	$(CLI) deploy

ftp-deploy:
	$(CLI) deploy

sftp-deploy:
	$(CLI) deploy

health:
	$(CLI) health

workflow-install:
	$(CLI) workflow:install --force
