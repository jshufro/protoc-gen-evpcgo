protos_rel = $(wildcard protos/*.proto)
protos = $(addprefix test/, $(protos_rel))
pwd = $(shell pwd)

all: clean test gopb abi/rocket_storage.go

.PHONY: clean gopb evpcgo
gopb: $(protos_rel)
	protoc --proto_path=.. --go_out=. $(protos) ../options.proto
evpcgo: $(protos_rel) protoc-gen-evpc
	PATH=${PATH}:$(pwd) protoc --proto_path=.. --evpcgo_out=. $(protos)
protoc-gen-evpc: ../main.go
	go build ..
test: main.go gopb evpcgo
	go build

clean:
	rm -f pb/*
