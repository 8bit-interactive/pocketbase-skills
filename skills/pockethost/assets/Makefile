# Compatibility Makefile for projects that install pocketbase-pockethost globally.
SHELL := /bin/sh

CLI ?= pocketbase-pockethost

.PHONY: check test deploy health install build workflow-install

install:
	$(CLI) pocketbase install

check:
	$(CLI) doctor

test:
	$(CLI) validate

build:
	@echo "No frontend build is required for a zero-build PocketBase site."

deploy:
	$(CLI) deploy

health:
	$(CLI) health

workflow-install:
	$(CLI) workflow install --force
