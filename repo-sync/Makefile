PLATFORM ?= linux/arm64

# Repo a subir (por defecto el directorio actual donde corres make)
REPO_DIR ?= $(PWD)

# MinIO via port-forward en tu Mac
MINIO_ENDPOINT ?= http://host.docker.internal:9000
MINIO_BUCKET ?= repos

REPO_NAME ?= $(notdir $(REPO_DIR))

.PHONY: build

build:
	docker build --platform $(PLATFORM) -t $(IMAGE) .
