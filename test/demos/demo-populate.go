package main

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jshufro/protoc-gen-evpcgo/test/abi"
)

func main() {
	// Initialize some stuff
	rocketStorageAddress := common.HexToAddress("0x1d8f8f00cfa6758d7bE78336684788Fb0ee0Fa46")
	rocketDAOProtocolSettingsDepositAddress := common.HexToAddress("0xac2245BE4C2C1E9752499Bcd34861B761d62fC27")

	client, err := ethclient.Dial("http://192.168.1.5:8545")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	// Create a struct with the addresses
	storageDetails := abi.NewStorage_Details(rocketDAOProtocolSettingsDepositAddress, rocketStorageAddress)
	// Bind it to an ethclient (ContractBackend)
	storageCaller, err := storageDetails.Bind(client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	// Create an empty struct
	storage := &abi.Storage{}
	// Populate it
	err = storageCaller.Populate(storage, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	fmt.Printf("struct contents: %+v\n", storage)
}
