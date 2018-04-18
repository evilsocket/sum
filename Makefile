all: server client

server: proto/sum.pb.go sumd

client: proto/sum_pb2.py 

sumd:
	@echo "Building sumd binary ..."
	@go build -o sumd .

proto/sum.pb.go:
	@echo "Generating Go protocol files ..."
	@protoc -I. --go_out=plugins=grpc:. proto/sum.proto

proto/sum_pb2.py:
	@echo "Generating Python protocol files ..."
	@python -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. proto/sum.proto
	@touch proto/__init__.py

clean:
	@rm -rf proto/*.go
	@rm -rf proto/*.py
	@rm -rf sumd
