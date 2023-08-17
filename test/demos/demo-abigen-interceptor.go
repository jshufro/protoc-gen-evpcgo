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
	rocketStorageAddress := common.HexToAddress("0x1d8f8f00cfa6758d7bE78336684788Fb0ee0Fa46")
	rocketDAOProtocolSettingsDepositAddress := common.HexToAddress("0xac2245BE4C2C1E9752499Bcd34861B761d62fC27")
	multicallerAddress := common.HexToAddress("0x5BA1e12693Dc8F9c48aAD8770482f4739bEeD696")

	client, err := ethclient.Dial("http://192.168.1.5:8545")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	// Create interceptor types from the generic NewInterceptor
	rsInterceptor, err := lib.NewInterceptor[*abi.RocketStorageCaller](rocketStorageAddress, abi.RocketStorageMetaData, abi.NewRocketStorageCaller)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	rdpsdInterceptor, err := lib.NewInterceptor[*abi.RocketDAOProtocolSettingsDepositCaller](rocketDAOProtocolSettingsDepositAddress, abi.RocketDAOProtocolSettingsDepositMetaData, abi.NewRocketDAOProtocolSettingsDepositCaller)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	// Create a multicaller
	mc, err := multicall.NewMultiCaller(client, multicallerAddress)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	// Define stack vars for the fields you want to populate
	var guardian common.Address
	var deployedStatus bool
	var depositEnabled bool

	// Create a call slice
	calls := make([]*lib.Call, 0)
	// Populate the call slice using the interceptor state machine
	err = rsInterceptor.Intercept(&calls, func() {
		rsInterceptor.SetDestination(&guardian)
		rsInterceptor.Contract().GetGuardian(nil)

		rsInterceptor.SetDestination(&deployedStatus)
		rsInterceptor.Contract().GetDeployedStatus(nil)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	// Extend the call slice using the interceptor state machine
	err = rdpsdInterceptor.Intercept(&calls, func() {
		rdpsdInterceptor.SetDestination(&depositEnabled)
		rdpsdInterceptor.Contract().GetDepositEnabled(nil)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	// Add the calls to the multicaller
	err = multicallWrapper(mc, calls)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	// Execute the multicaller
	_, err = mc.FlexibleCall(true, &bind.CallOpts{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	fmt.Println(fmt.Sprint("guardian: ", guardian))
	fmt.Println(fmt.Sprint("DeployedStatus: ", deployedStatus))
	fmt.Println(fmt.Sprint("DepositEnabled: ", depositEnabled))
}
