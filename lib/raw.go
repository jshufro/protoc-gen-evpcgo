package lib

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type contextKey int

const callMsgKey = contextKey(0)

type Call struct {
	Address     *common.Address
	Abi         *abi.ABI
	CallData    func() ([]byte, error)
	Method      string
	Destination interface{}
}

// An interceptor lets you call abigen-created type-safe functions, but without actually
// calling a live backend- instead, we let abigen encode the calldata and intercept it
// by passing a false ContractCaller

type ABIMetaData interface {
	GetAbi() (*abi.ABI, error)
}

type callable interface {
	Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error
}

type Interceptor[T any] struct {
	contractAddress common.Address
	abi             *abi.ABI
	contract        T

	lock     sync.Mutex
	callInfo struct {
		seq uint
		err error
		dst interface{}

		out *[]*Call
	}
}

func (i *Interceptor[T]) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	dst := i.callInfo.out

	method, _ := i.abi.MethodById(call.Data)

	out := new(Call)
	out.Address = call.To
	out.Abi = i.abi
	out.CallData = func() ([]byte, error) {
		return call.Data, nil
	}
	out.Destination = i.callInfo.dst
	out.Method = method.Name
	*dst = append(*dst, out)

	return nil, nil
}

func (i *Interceptor[T]) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {

	return nil, fmt.Errorf("CodeAt unimplemented")
}

func (i *Interceptor[T]) ensureLocked() error {
	// Make sure we hold the lock
	if i.lock.TryLock() == true {
		i.lock.Unlock()
		return fmt.Errorf("invalid operation- Contract() and SetDestination() must be called only in Intercept()'s callback")
	}

	return nil
}

func (i *Interceptor[T]) Contract() T {
	if err := i.ensureLocked(); err != nil {
		i.callInfo.err = err
	}

	if i.callInfo.seq != 1 {
		i.callInfo.err = fmt.Errorf("Contract() must only be called after each SetDestination() call")
	}

	i.callInfo.seq = 0
	return i.contract
}

func (i *Interceptor[T]) SetDestination(dst interface{}) {
	if err := i.ensureLocked(); err != nil {
		i.callInfo.err = err
	}

	if i.callInfo.seq != 0 {
		i.callInfo.err = fmt.Errorf("SetDestination() must only be called before each Contract() call")
	}

	i.callInfo.dst = dst
	i.callInfo.seq = 1
}

func (i *Interceptor[T]) Intercept(out *[]*Call, cb func()) error {

	i.lock.Lock()
	defer i.lock.Unlock()

	i.callInfo.out = out
	cb()
	if i.callInfo.err != nil {
		return i.callInfo.err
	}
	return nil
}

func NewInterceptor[T any](address common.Address,
	md ABIMetaData,
	constructor func(common.Address, bind.ContractCaller) (T, error)) (*Interceptor[T], error) {

	parsedAbi, err := md.GetAbi()
	if err != nil {
		return nil, err
	}

	out := &Interceptor[T]{
		contractAddress: address,
		abi:             parsedAbi,
	}

	contract, err := constructor(address, out)
	out.contract = contract

	return out, nil
}
