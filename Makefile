all: server clients

server: deps proto/sum.pb.go sumd

clients: clients/python/proto/sum_pb2.py clients/php/Sum 


sumd:
	@echo "Building sumd binary ..."
	@go build -o sumd .

deps:
	@dep ensure

proto/sum.pb.go:
	@echo "Generating Go protocol files ..."
	@/opt/grpc/bins/opt/protobuf/protoc -I. --go_out=plugins=grpc:. proto/sum.proto

clients/python/proto/sum_pb2.py:
	@echo "Generating Python protocol files ..."
	@python -m grpc_tools.protoc \
		-Iproto \
		--python_out=clients/python/proto \
		--grpc_python_out=clients/python/proto \
		proto/sum.proto

clients/php/Sum:
	@echo "Generating PHP protocol files ..."
	@/opt/grpc/bins/opt/protobuf/protoc -I. --proto_path=proto \
		--php_out=clients/php \
		--grpc_out=clients/php \
		--plugin=protoc-gen-grpc=/opt/grpc/bins/opt/grpc_php_plugin \
		proto/sum.proto

test:
	@go test ./...

benchmark:
	@go test -run=xxx -bench=. ./...

clean:
	@rm -rf proto/*.go
	@rm -rf clients/python/proto/sum_*.py
	@rm -rf clients/php/Sum
	@rm -rf clients/php/GPBMetadata
	@rm -rf sumd

run:
	@clear 
	@make clean
	@make 
	@clear 
	@sudo rm -rf /var/lib/sumd
	@sudo mkdir -p /var/lib/sumd/data
	@sudo mkdir -p /var/lib/sumd/oracles
	@sudo ./sumd -cpu-profile cpu.profile -mem-profile mem.profile
