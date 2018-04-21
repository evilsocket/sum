all: sumd clients

server_deps: deps proto/sum.pb.go

clients: clients/python/proto/sum_pb2.py clients/php/Sum 

deps:
	@dep ensure

sumd: server_deps
	@echo "Building sumd binary ..."
	@go build -o sumd .

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

benchmark: server_deps
	@echo "Running benchmarks ..."
	@go test -run=xxx -bench=. ./...

test: server_deps
	@echo "Running tests ...\n"
	@go test ./... -v -coverprofile=coverage.profile -covermode=atomic

html_coverage: test
	@echo "\nGenerating code coverage report to coverage.profile.html ..."
	@go tool cover -html=coverage.profile -o coverage.profile.html
	
codecov: test
	@echo "Uploading code coverage profile to codecov.io ..."
	@echo $(curl -s https://codecov.io/bash)

clean:
	@echo "Cleaning ..."
	@rm -rf proto/*.go
	@rm -rf clients/python/proto/sum_*.py
	@rm -rf clients/php/Sum
	@rm -rf clients/php/GPBMetadata
	@rm -rf sumd
	@rm -rf *.profile
	@rm -rf *.profile.html

reset_env: clean
	@echo "Resetting environment ..."
	@clear 
	@sudo rm -rf /var/lib/sumd
	@sudo mkdir -p /var/lib/sumd/data
	@sudo mkdir -p /var/lib/sumd/oracles

profile: reset_env sumd
	@clear
	@sudo ./sumd -cpu-profile cpu.profile -mem-profile mem.profile

run: reset_env sumd
	@clear
	@sudo ./sumd
