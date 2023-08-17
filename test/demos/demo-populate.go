package main

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jshufro/protoc-gen-evpcgo/test/abi"
)

type addressProvider struct {
	storage         common.Address
	settingsDeposit common.Address
}

func (a *addressProvider) RocketStorageAddress() (*common.Address, error) {
	return &a.storage, nil
}

func (a *addressProvider) RocketDAOProtocolSettingsDepositAddress() (*common.Address, error) {
	return &a.settingsDeposit, nil
}

func main() {
	// Initialize some stuff
	addresser := &addressProvider{
		storage:         common.HexToAddress("0x1d8f8f00cfa6758d7bE78336684788Fb0ee0Fa46"),
		settingsDeposit: common.HexToAddress("0xac2245BE4C2C1E9752499Bcd34861B761d62fC27"),
	}

	client, err := ethclient.Dial("http://192.168.1.5:8545")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	// Create a struct with the addresses
	w, err := abi.NewStorageWriter()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	// Bind it to an ethclient (ContractBackend)
	bw, err := w.Bind(client, addresser)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	// Create an empty struct
	storage := &abi.Storage{}
	// Populate it
	err = bw.Populate(storage, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	fmt.Printf("struct contents: %+v\n", storage)

	// Alternatively, let the library do the binding ad-hoc
	storage = &abi.Storage{}
	err = w.Populate(storage, client, addresser, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	fmt.Printf("struct contents: %+v\n", storage)

	// We can also populate one field at a time
	storage = &abi.Storage{}
	err = bw.PopulateGuardian(storage, nil)
	fmt.Printf("guardian contents: %+v\n", storage.Guardian)

	// We can also populate one field at a time without binding first
	storage = &abi.Storage{}
	err = w.PopulateGuardian(storage, client, addresser, nil)
	fmt.Printf("guardian contents: %+v\n", storage.Guardian)

}
