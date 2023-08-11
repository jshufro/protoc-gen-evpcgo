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

	// abigen generates bindings at a contract level
	rs, err := abi.NewRocketStorage(rocketStorageAddress, client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	rdpsd, err := abi.NewRocketDAOProtocolSettingsDeposit(rocketDAOProtocolSettingsDepositAddress, client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	// abigen lets you query one function at a time, but doesn't intuitively support multicall
	guardian, err := rs.GetGuardian(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	deployedStatus, err := rs.GetDeployedStatus(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	depositEnabled, err := rdpsd.GetDepositEnabled(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	// variables are now populated
	fmt.Println(fmt.Sprint("guardian: ", guardian))
	fmt.Println(fmt.Sprint("DeployedStatus: ", deployedStatus))
	fmt.Println(fmt.Sprint("DepositEnabled: ", depositEnabled))
}
