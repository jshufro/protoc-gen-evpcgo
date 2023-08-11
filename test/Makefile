protos_rel = $(wildcard protos/*.proto)
protos = $(addprefix test/, $(protos_rel))

hardhat_artifacts = $(shell find rocketpool/artifacts/contracts/contract -wholename "*.sol/*.json" -not -name "*.dbg.json")
abis = $(patsubst %.json,abi/%.abi, $(notdir $(hardhat_artifacts)))
abigo = $(patsubst %.abi,%.go, $(abis))

pwd = $(shell pwd)

all: $(abigo) clean test gopb

.PHONY: clean gopb

gopb: $(protos_rel)
	protoc --proto_path=.. --go_out=. $(protos) ../options.proto

abi/%_evpc.go: protoc-gen-evpc $(protos_rel)
	PATH=${PATH}:$(pwd) protoc --proto_path=.. --evpcgo_out=. $(protos)

protoc-gen-evpc: ../main.go
	go build ..

vpath %.json $(shell find rocketpool/artifacts/contracts/contract -type d -name "*.sol")
abi/%.abi: %.json
	cat $? | jq .abi > $@

abi/%.go: abi/%.abi
	abigen --abi $? --pkg abi --type $(patsubst %.abi,%,$(notdir $?)) --out $@

test: main.go gopb abi/storage_evpc.go
	go build

clean:
	rm -rf pb/*
