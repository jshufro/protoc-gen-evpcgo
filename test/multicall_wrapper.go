package main

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rocket-pool/rocketpool-go/rocketpool"
	"github.com/rocket-pool/rocketpool-go/utils/multicall"
)

// We're going to make multicall a ContractCaller, that way we can bind contracts to it directly.

// This means we can use abigen's typesafe getters with multicall.

type ContextKey string

const ABIKey = ContextKey("_abi")
const MethodKey = ContextKey("_method")
const DstKey = ContextKey("_dst")

type errorQueued string

func (q errorQueued) Error() string {
	return string(q)
}

const Queued = errorQueued("call queued")

type OmniCaller struct {
	*multicall.MultiCaller
	blockNumber *big.Int
}

func NewOmniCaller(client rocketpool.ExecutionClient, multicallerAddress common.Address) (*OmniCaller, error) {
	var err error

	out := new(OmniCaller)
	out.MultiCaller, err = multicall.NewMultiCaller(client, multicallerAddress)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (o *OmniCaller) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	return o.MultiCaller.Client.CodeAt(ctx, contract, blockNumber)
}

func (o *OmniCaller) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	if o.blockNumber == nil {
		o.blockNumber = blockNumber
	} else if o.blockNumber.Cmp(blockNumber) != 0 {
		// it's ignored anyway, but we will cry about it here all the same
		return nil, fmt.Errorf("all calls in a multicaller must have the same block number")
	}

	abi, ok := ctx.Value(ABIKey).(*abi.ABI)
	if !ok {
		return nil, fmt.Errorf("you must populate the abi value in request context")
	}

	method, ok := ctx.Value(MethodKey).(string)
	if !ok {
		return nil, fmt.Errorf("you must populate the method value in request context")
	}

	dst, ok := ctx.Value(DstKey).(interface{})
	if !ok {
		return nil, fmt.Errorf("you must populate the method value in request context")
	}

	// Add the call to the multicaller...
	o.MultiCaller.Calls = append(o.MultiCaller.Calls, multicall.Call{
		Target:   *call.To,
		CallData: call.Data,
		UnpackFunc: func(rawData []byte) error {
			return abi.UnpackIntoInterface(dst, method, rawData)
		},
	})
	return nil, Queued
}
