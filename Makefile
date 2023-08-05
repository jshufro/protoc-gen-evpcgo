all: pb/options.pb

pb/options.pb: options.proto
	protoc --proto_path=. --go_out=. options.proto
