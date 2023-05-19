PROTOC_IMAGE_NAME=registry.devops.rivtower.com/cita-cloud/protoc
PROTOC_IMAGE_VERSION=3.19.1

protoc-image-build:
	docker build --platform linux/arm64 --network=host -t $(PROTOC_IMAGE_NAME):$(PROTOC_IMAGE_VERSION) -f ./Dockerfile-protoc-3.19.1 .

protoc-image-push:
	docker push $(PROTOC_IMAGE_NAME):$(PROTOC_IMAGE_VERSION)

grpc-code-generate:
	docker run -v $(PWD):/src -e GO111MODULE=on $(PROTOC_IMAGE_NAME):$(PROTOC_IMAGE_VERSION) /bin/bash ./grpc-code-generate.sh

compile:
	protoc api/v1/*.proto \
		--go_out=. \
		--go_out=paths=source_relative \
		--proto_path=.