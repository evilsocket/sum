.PHONY: all clients godep golint gomegacheck deps test codecov html_coverage benchmark
.PHONY: clean reset_env profile run build_docker run_docker pycli phpcli

all: sumd clients

server_deps: deps proto/sum.pb.go

clients: pycli phpcli 

godep:
	@go get -u github.com/golang/dep/...

deps: godep golint gomegacheck
	@dep ensure

proto/sum.pb.go:
	@/opt/grpc/bins/opt/protobuf/protoc -I. --go_out=plugins=grpc:. proto/sum.proto

sumd: server_deps
	@go build -o sumd .

clean:
	@rm -rf proto/*.go
	@rm -rf clients/python/proto/sum_*.py
	@rm -rf clients/php/Sum
	@rm -rf clients/php/GPBMetadata
	@rm -rf sumd
	@rm -rf *.profile
	@rm -rf *.profile.html

reset_env: clean
	@sudo rm -rf /var/lib/sumd
	@sudo mkdir -p /var/lib/sumd/data
	@sudo mkdir -p /var/lib/sumd/oracles

run: reset_env sumd
	@sudo ./sumd

#
# Testing and benchmarking
#
golint:
	@go get github.com/golang/lint/golint

gomegacheck:
	@go get honnef.co/go/tools/cmd/megacheck

# Go 1.9 doesn't support test coverage on multiple packages, while
# Go 1.10 does, let's keep it 1.9 compatible in order not to break
# travis
define testPackage
	@go vet ./$(1)
	@golint -set_exit_status ./$(1)
	@megacheck ./$(1)
	@go test -v ./$(1) -coverprofile=$(1).profile
	@tail -n +2 $(1).profile >> coverage.profile && rm $(1).profile
endef

test: server_deps gomegacheck golint
	@echo "mode: set" > coverage.profile
	$(call testPackage,service)
	$(call testPackage,storage)
	$(call testPackage,wrapper)
	
codecov: test
	@echo $(curl -s https://codecov.io/bash)

html_coverage: test
	@go tool cover -html=coverage.profile -o coverage.profile.html

profile: reset_env sumd
	@sudo ./sumd -cpu-profile cpu.profile -mem-profile mem.profile

benchmark: server_deps
	@go test ./... -v -run=doNotRunTests -bench=. -benchmem

#
# Docker stuff
#
docker:
	@docker build -t sumd:latest .

run_docker: docker
	@docker run -it -p 50051:50051 sumd:latest

#
# Client code generation related stuff.
#
pycli:
	@python -m grpc_tools.protoc \
		-Iproto \
		--python_out=clients/python/proto \
		--grpc_python_out=clients/python/proto \
		proto/sum.proto

phpcli:
	@/opt/grpc/bins/opt/protobuf/protoc -I. --proto_path=proto \
		--php_out=clients/php \
		--grpc_out=clients/php \
		--plugin=protoc-gen-grpc=/opt/grpc/bins/opt/grpc_php_plugin \
		proto/sum.proto
