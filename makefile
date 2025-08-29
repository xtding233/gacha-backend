# Where your proto files live
PROTO_DIR=internal/proto
# Where generated Go code should go
GEN_DIR=gen

# Find all .proto files under internal/proto
PROTO_FILES := $(shell find $(PROTO_DIR) -name '*.proto')

.PHONY: proto clean

proto:
	@mkdir -p $(GEN_DIR)
	protoc -I $(PROTO_DIR) \
	  --go_out=$(GEN_DIR) --go_opt=paths=source_relative \
	  --go-grpc_out=$(GEN_DIR) --go-grpc_opt=paths=source_relative \
	  $(PROTO_FILES)

clean:
	rm -rf $(GEN_DIR)
