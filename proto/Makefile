.PHONY: all generate go python clean
PROTO_DIR := ./
PROTO_FILES = $(shell find $(PROTO_DIR) -name '*.proto')

GO_OUT_DIR := ../shared/go/v1
PYTHON_OUT_DIR := ../shared/python/v1

all: generate
generate: go python

go:
	@echo "GENERATING GO PROTOBUF..."
	@mkdir -p $(GO_OUT_DIR)
	@protoc -I=$(PROTO_DIR) \
			--go_out=$(GO_OUT_DIR) --go_opt=paths=source_relative \
			--go-grpc_out=$(GO_OUT_DIR) --go-grpc_opt=paths=source_relative \
			$(PROTO_FILES)
	@echo "GO PROTOBUF (1.25.6) GENERATED SUCCESSFULLY."

python:
	@echo "GENERATING PYTHON PROTOBUF..."
	@mkdir -p $(PYTHON_OUT_DIR)
	@python3 -m grpc_tools.protoc -I=$(PROTO_DIR) \
			--python_out=$(PYTHON_OUT_DIR) \
			--grpc_python_out=$(PYTHON_OUT_DIR) \
			$(PROTO_FILES)
	@echo "PYTHON PROTOBUF (3.12) GENERATED SUCCESSFULLY."

clean:
	@echo "CLEANING GENERATED FILES..."
	@rm -rf $(GO_OUT_DIR)/* $(PYTHON_OUT_DIR)/*
	@echo "CLEANUP COMPLETED."