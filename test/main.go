package main

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/ethclient"
	_ "github.com/jshufro/protoc-gen-evpcgo/test/abi"
	"github.com/jshufro/protoc-gen-evpcgo/test/pb"
)

func main() {
	client, err := ethclient.Dial("http://192.168.1.5:8545")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%w", err)
		return
	}

	// Get rocket storage settings
	storage := &pb.Storage{}
	pb.PopulateStorage(storage, client, nil)

	fmt.Printf("%+v\n", storage)
}
