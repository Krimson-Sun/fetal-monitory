# ---- paths ----
PROTO_DIR   := proto
OUT_DIR     := $(PROTO_DIR)      # генерим рядом с .proto
LOCAL_BIN   := $(CURDIR)/bin
PROTO_FILES := $(shell find $(PROTO_DIR) -type f -name "*.proto")

.PHONY: tools gen clean

tools:
	@mkdir -p $(LOCAL_BIN)
	GOBIN=$(LOCAL_BIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	GOBIN=$(LOCAL_BIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

gen: tools
	@echo ">> generating into $(OUT_DIR)"
	@PATH=$(LOCAL_BIN):$$PATH protoc \
	  -I $(PROTO_DIR) \
	  --go_out=$(OUT_DIR) --go_opt=paths=source_relative \
	  --go-grpc_out=$(OUT_DIR) --go-grpc_opt=paths=source_relative \
	  $(PROTO_FILES)
	@go mod tidy

clean:
	@rm -rf $(LOCAL_BIN)
	@find $(PROTO_DIR) -type f \( -name "*_grpc.pb.go" -o -name "*.pb.go" \) -delete
