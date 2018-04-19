all: server client

server: deps proto/sum.pb.go sumd

client: client/proto/sum_pb2.py 

sumd:
	@echo "Building sumd binary ..."
	@go build -o sumd .

deps:
	@dep ensure

proto/sum.pb.go:
	@echo "Generating Go protocol files ..."
	@protoc -I. --go_out=plugins=grpc:. proto/sum.proto

client/proto/sum_pb2.py:
	@echo "Generating Python protocol files ..."
	@python -m grpc_tools.protoc -Iproto --python_out=client/proto --grpc_python_out=client/proto proto/sum.proto

clean:
	@rm -rf proto/*.go
	@rm -rf client/proto/sum_*.py
	@rm -rf sumd

run:
	@clear 
	@make clean
	@make 
	@clear 
	@sudo rm -rf /var/lib/sumd
	@sudo mkdir -p /var/lib/sumd/data
	@sudo mkdir -p /var/lib/sumd/oracles
	@sudo ./sumd
