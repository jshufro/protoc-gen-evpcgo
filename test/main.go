package main

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	_ "github.com/jshufro/protoc-gen-evpcgo/test/abi"
	"github.com/jshufro/protoc-gen-evpcgo/test/pb"
)

func main() {
	rocketStorageAddress := common.HexToAddress("0x1d8f8f00cfa6758d7bE78336684788Fb0ee0Fa46")
	rocketDAOProtocolSettingsDepositAddress := common.HexToAddress("0xac2245BE4C2C1E9752499Bcd34861B761d62fC27")

	client, err := ethclient.Dial("http://192.168.1.5:8545")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%w", err)
		return
	}

	storageDetails := pb.NewStorage_Details(rocketStorageAddress, rocketDAOProtocolSettingsDepositAddress)

	storageCaller, err := storageDetails.Bind(client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%w", err)
	}

	storage := &pb.Storage{}
	storageCaller.Populate(storage, nil)

	fmt.Printf("%+v\n", storage)
}
