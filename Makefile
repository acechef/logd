PROTOC_IMAGE_NAME=registry.devops.rivtower.com/cita-cloud/protoc
PROTOC_IMAGE_VERSION=3.19.1

CONFIG_PATH=${HOME}/.logd

.PHONY: init
init:
	mkdir -p ${CONFIG_PATH}

.PHONY: gencert
gencert:
	cfssl gencert \
		-initca test/ca-csr.json | cfssljson -bare ca
	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=server \
		test/server-csr.json | cfssljson -bare server
	cfssl gencert \
    	-ca=ca.pem \
    	-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=client \
		-cn="root" \
		test/client-csr.json | cfssljson -bare root-client
	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=client \
		-cn="nobody" \
		test/client-csr.json | cfssljson -bare nobody-client
	mv *.pem *.csr ${CONFIG_PATH}

${CONFIG_PATH}/model.conf:
	cp test/model.conf ${CONFIG_PATH}/model.conf

${CONFIG_PATH}/policy.csv:
	cp test/policy.csv ${CONFIG_PATH}/policy.csv

.PHONY: test
test: ${CONFIG_PATH}/policy.csv ${CONFIG_PATH}/model.conf
	go test -race ./...

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