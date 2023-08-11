package main

import (
	"context"
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
func multicallWrapper(mc *multicall.MultiCaller, calls []*lib.Call) {
	for _, call := range calls {
		c := call // Goddamn golang closures are annoying. Don't close over the iterator.
		mc.Calls = append(mc.Calls, multicall.Call{
			Target:   *c.Address,
			CallData: c.CallData,
			UnpackFunc: func(rawData []byte) error {
				err := c.Abi.UnpackIntoInterface(c.Destination, c.Method, rawData)
				if err != nil {
					return fmt.Errorf("error unpacking data %v in multicall result for call %+v: %v", rawData, call, err)
				}
				return nil
			},
		})
	}
}

func main() {
	rocketStorageAddress := common.HexToAddress("0x1d8f8f00cfa6758d7bE78336684788Fb0ee0Fa46")
	rocketDAOProtocolSettingsDepositAddress := common.HexToAddress("0xac2245BE4C2C1E9752499Bcd34861B761d62fC27")
	multicallerAddress := common.HexToAddress("0x5BA1e12693Dc8F9c48aAD8770482f4739bEeD696")

	client, err := ethclient.Dial("http://192.168.1.5:8545")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	// Gross hacky thing for posterity
	// =============================================================
	{
		mc, err := NewOmniCaller(client, multicallerAddress)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return
		}

		rs, err := abi.NewRocketStorageCaller(rocketStorageAddress, mc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return
		}
		rdpsd, err := abi.NewRocketDAOProtocolSettingsDepositCaller(rocketDAOProtocolSettingsDepositAddress, mc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return
		}

		rsAbi, _ := abi.RocketStorageMetaData.GetAbi()
		guardian := new(common.Address)
		ctx := context.Background()
		ctx = context.WithValue(ctx, ABIKey, rsAbi)
		ctx = context.WithValue(ctx, MethodKey, "getGuardian")
		ctx = context.WithValue(ctx, DstKey, guardian)
		_, err = rs.GetGuardian(&bind.CallOpts{
			Context: ctx,
		})
		if err != Queued {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return
		}

		deployedStatus := new(bool)
		ctx = context.Background()
		ctx = context.WithValue(ctx, ABIKey, rsAbi)
		ctx = context.WithValue(ctx, MethodKey, "getDeployedStatus")
		ctx = context.WithValue(ctx, DstKey, deployedStatus)
		_, err = rs.GetDeployedStatus(&bind.CallOpts{
			Context: ctx,
		})
		if err != Queued {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return
		}

		rdpsdAbi, _ := abi.RocketDAOProtocolSettingsDepositMetaData.GetAbi()
		depositEnabled := new(bool)
		ctx = context.Background()
		ctx = context.WithValue(ctx, ABIKey, rdpsdAbi)
		ctx = context.WithValue(ctx, MethodKey, "getDepositEnabled")
		ctx = context.WithValue(ctx, DstKey, depositEnabled)
		_, err = rdpsd.GetDepositEnabled(&bind.CallOpts{
			Context: ctx,
		})
		if err != Queued {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return
		}

		// Omnicall is primed, now execute its underlying mc
		mc.MultiCaller.FlexibleCall(false, &bind.CallOpts{})
		fmt.Println(fmt.Sprint("guardian: ", guardian))
		fmt.Println(fmt.Sprint("DeployedStatus: ", *deployedStatus))
		fmt.Println(fmt.Sprint("DepositEnabled: ", *depositEnabled))
	}
}
