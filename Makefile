all: sumd build_docker clients

server_deps: deps proto/sum.pb.go

clients: clients/python/proto/sum_pb2.py clients/php/Sum 

godep:
	@go get -u github.com/golang/dep/...

deps: godep
	@dep ensure

build_docker:
	@docker build -t sumd:latest .

run_docker: build_docker
	@docker run -it -p 50051:50051 sumd:latest

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
	@go test ./... -v -run=doNotRunTests -bench=.

# go 1.9 doesn't support test coverage on multiple packages, while
# go 1.10 does, let's keep it 1.9 compatible in order not to break
# travis
service.profile:
	@go test ./service -coverprofile=service.profile

storage.profile:
	@go test ./storage -coverprofile=storage.profile

wrapper.profile:
	@go test ./wrapper -coverprofile=wrapper.profile

coverage.profile: service.profile storage.profile wrapper.profile
	@echo "mode: set" > coverage.profile
	@tail -n +2 service.profile >> coverage.profile && rm service.profile
	@tail -n +2 storage.profile >> coverage.profile && rm storage.profile
	@tail -n +2 wrapper.profile >> coverage.profile && rm wrapper.profile

test: server_deps coverage.profile 

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
