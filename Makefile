SHELL := bash
.HONY: all clients godep golint deps test codecov html_coverage benchmark
.PHONY: clean reset_env profile run build_docker run_docker sumpy sumphp

#
# Config
#
GRPC_PATH=/opt/grpc/bins/opt
GRPC_PHP_PLUGIN=${GRPC_PATH}/grpc_php_plugin
GRPC_PROTOC=/opt/google/protoc/bin/protoc

SUMPY_PATH=${HOME}/lab/sumpy
SUMPHP_PATH=${HOME}/lab/sumphp

SUMD_DATAPATH=/tmp/sumd

PACKAGES=node/storage node/wrapper node/service master

#
# Main actions
#
all: sumd sumcli sumcluster

server_deps: deps proto/sum.pb.go
client_deps: deps proto/sum.pb.go

godep:
	@go get -u github.com/golang/dep/...

deps: godep golint
	@dep ensure

proto/sum.pb.go:
	@${GRPC_PROTOC} -I. --go_out=plugins=grpc:. proto/sum.proto

sumcli: client_deps
	@mkdir -p dist
	@go build -o dist/sumcli cmd/sumcli/*.go

sumcluster: 
	@mkdir -p dist
	@go build -o dist/sumcluster cmd/sumcluster/*.go

sumd: server_deps sumcli sumcluster
	@mkdir -p dist
	@go build -o dist/sumd cmd/sumd/*.go

clean:
	@rm -rf dist
	@rm -rf proto/sum.pb.go
	@rm -rf *.profile
	@rm -rf *.profile.html
	@rm -rf "${SUMD_DATAPATH}"

reset_env: clean
	@mkdir -p "${SUMD_DATAPATH}"
	@mkdir -p "${SUMD_DATAPATH}/data"
	@mkdir -p "${SUMD_DATAPATH}/oracles"

install_certificate:
	@mkdir -p /etc/sumd/creds
	@openssl req -x509 -newkey rsa:4096 -keyout /etc/sumd/creds/key.pem -out /etc/sumd/creds/cert.pem -days 365 -nodes -subj '/CN=localhost'

install:
	@mkdir -p /var/lib/sumd/data
	@mkdir -p /var/lib/sumd/oracles
	@cp dist/{sumd,sumcli,sumcluster} /usr/local/bin/

install_service: install
	@cp sumd.service /etc/systemd/system/
	@systemctl daemon-reload

#
# Testing and benchmarking
#
golint:
	@go get github.com/golang/lint/golint

lint: golint
	@for pkg in $(PACKAGES); do \
		go vet ./$$pkg ; \
		golint -set_exit_status ./$$pkg ; \
	done

# Go 1.9 doesn't support test coverage on multiple packages, while
# Go 1.10 does, let's keep it 1.9 compatible in order not to break
# travis
test: server_deps 
	@echo "mode: atomic" > coverage.profile
	@for pkg in $(PACKAGES); do \
		go test -race ./$$pkg -coverprofile=$$pkg.profile -covermode=atomic; \
		tail -n +2 $$pkg.profile >> coverage.profile && rm $$pkg.profile ; \
	done
	
codecov: test
	@bash <(curl -s https://codecov.io/bash)

html_coverage: test
	@go tool cover -html=coverage.profile -o coverage.profile.html

profile: reset_env sumd
	@./sumd -datapath "${SUMD_DATAPATH}" -cpu-profile cpu.profile -mem-profile mem.profile

benchmark: server_deps
	@go test ./... -v -run=doNotRunTests -bench=. -benchmem

follow:
	@pidstat --human -l -u -p `pidof sumd` 1

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
clients: sumcli sumpy sumphp 

sumpy:
	@python -m grpc_tools.protoc \
		-Iproto \
		--python_out=${SUMPY_PATH}/sumpy/proto \
		--grpc_python_out=${SUMPY_PATH}/sumpy/proto \
		proto/sum.proto
	@git --git-dir=${SUMPY_PATH}/.git --work-tree=${SUMPY_PATH} status

sumphp:
	@${GRPC_PROTOC} -I. --proto_path=proto \
		--php_out=${SUMPHP_PATH} \
		--grpc_out=${SUMPHP_PATH} \
		--plugin=protoc-gen-grpc=${GRPC_PHP_PLUGIN} \
		proto/sum.proto
	@git --git-dir=${SUMPHP_PATH}/.git --work-tree=${SUMPHP_PATH} status
