package main

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jshufro/protoc-gen-evpcgo/lib"
	"github.com/jshufro/protoc-gen-evpcgo/test/abi"
	"github.com/rocket-pool/rocketpool-go/utils/multicall"
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

// Utility function to translate the library Call format to multicaller's expected call format
func multicallWrapper(mc *multicall.MultiCaller, calls []*lib.Call) error {
	for _, call := range calls {
		c := call // Goddamn golang closures are annoying. Don't close over the iterator.
		callData, err := c.CallData()
		if err != nil {
			return fmt.Errorf("Error getting CallData for %s: %v", call.Method, err)
		}
		mc.Calls = append(mc.Calls, multicall.Call{
			Target:   *c.Address,
			CallData: callData,
			UnpackFunc: func(rawData []byte) error {
				err := c.Abi.UnpackIntoInterface(c.Destination, c.Method, rawData)
				if err != nil {
					return fmt.Errorf("error unpacking data %v in multicall result for call %+v: %v", rawData, call, err)
				}
				return nil
			},
		})
	}
	return nil
}

func main() {
	// Initialize some stuff
	addresser := &addressProvider{
		storage:         common.HexToAddress("0x1d8f8f00cfa6758d7bE78336684788Fb0ee0Fa46"),
		settingsDeposit: common.HexToAddress("0xac2245BE4C2C1E9752499Bcd34861B761d62fC27"),
	}
	multicallerAddress := common.HexToAddress("0x5BA1e12693Dc8F9c48aAD8770482f4739bEeD696")

	client, err := ethclient.Dial("http://192.168.1.5:8545")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	// Initialize a multicaller normally
	mc, err := multicall.NewMultiCaller(client, multicallerAddress)
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

	// Create the raw writer
	rw, err := w.Raw(addresser)

	// Create an empty struct
	storage := &abi.Storage{}
	// Get the calls needed to populate it
	calls := rw.AllCalls(storage)

	// Add them to the multicaller
	err = multicallWrapper(mc, calls)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	fmt.Printf("struct contents before mc.FlexibleCall: %+v\n", storage)

	// Execute the multicall
	_, err = mc.FlexibleCall(true, &bind.CallOpts{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	fmt.Printf("struct contents after mc.FlexibleCall: %+v\n", storage)

	// Create an empty struct
	storage = &abi.Storage{}
	fmt.Printf("struct contents before mc.FlexibleCall: %+v\n", storage)
	calls = make([]*lib.Call, 0)
	// Add calls piecemeal
	calls = append(calls, rw.Guardian(storage))
	calls = append(calls, rw.DeployedStatus(storage))
	calls = append(calls, rw.DepositEnabled(storage))

	// Add them to the multicaller
	err = multicallWrapper(mc, calls)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	// Execute the multicall
	_, err = mc.FlexibleCall(true, &bind.CallOpts{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	fmt.Printf("struct contents after mc.FlexibleCall: %+v\n", storage)
}
