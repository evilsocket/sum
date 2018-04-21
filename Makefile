.PHONY: all clients godep golint gomegacheck deps test codecov html_coverage benchmark
.PHONY: clean reset_env profile run build_docker run_docker pycli phpcli

GRPC_PATH=/opt/grpc/bins/opt
GRPC_PHP_PLUGIN=${GRPC_PATH}/grpc_php_plugin
GRPC_PROTOC=${GRPC_PATH}/protobuf/protoc

SUMD_DATAPATH=/tmp/sumd

#
# Main actions
#
all: sumd clients test codecov benchmark docker

server_deps: deps proto/sum.pb.go

clients: pycli phpcli 

godep:
	@go get -u github.com/golang/dep/...

deps: godep golint gomegacheck
	@dep ensure

proto/sum.pb.go:
	@${GRPC_PROTOC} -I. --go_out=plugins=grpc:. proto/sum.proto

sumd: server_deps
	@go build -o sumd .

run: reset_env sumd
	@./sumd -datapath "${SUMD_DATAPATH}"

clean:
	@rm -rf proto/*.go
	@rm -rf clients/python/proto/sum_*.py
	@rm -rf clients/php/Sum
	@rm -rf clients/php/GPBMetadata
	@rm -rf sumd
	@rm -rf *.profile
	@rm -rf *.profile.html
	@rm -rf "${SUMD_DATAPATH}"

reset_env: clean
	@mkdir -p "${SUMD_DATAPATH}"
	@mkdir -p "${SUMD_DATAPATH}/data"
	@mkdir -p "${SUMD_DATAPATH}/oracles"

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
	@go test -v -race ./$(1) -coverprofile=$(1).profile
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
	@./sumd -datapath "${SUMD_DATAPATH}" -cpu-profile cpu.profile -mem-profile mem.profile

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
	@${GRPC_PROTOC} -I. --proto_path=proto \
		--php_out=clients/php \
		--grpc_out=clients/php \
		--plugin=protoc-gen-grpc=${GRPC_PHP_PLUGIN} \
		proto/sum.proto
