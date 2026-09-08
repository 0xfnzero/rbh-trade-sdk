package pons

import (
	"context"
	"errors"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type blockNumberReader interface {
	BlockNumber(context.Context) (uint64, error)
}

func (c *Client) factory() *bind.BoundContract {
	if c == nil || c.addresses.Factory == (common.Address{}) {
		return nil
	}
	return bind.NewBoundContract(c.addresses.Factory, factoryContractABI, c.caller, nil, nil)
}

func (c *Client) curve(address common.Address) *bind.BoundContract {
	if c == nil || address == (common.Address{}) {
		return nil
	}
	return bind.NewBoundContract(address, curveContractABI, c.caller, c.transactor, c.filterer)
}

func (c *Client) escrow() *bind.BoundContract {
	if c == nil || c.addresses.FeeEscrow == (common.Address{}) {
		return nil
	}
	return bind.NewBoundContract(c.addresses.FeeEscrow, escrowContractABI, c.caller, c.transactor, c.filterer)
}

func (c *Client) token(address common.Address) *bind.BoundContract {
	if c == nil || address == (common.Address{}) {
		return nil
	}
	return bind.NewBoundContract(address, tokenContractABI, c.caller, c.transactor, c.filterer)
}

func (c *Client) vault() *bind.BoundContract {
	if c == nil || c.addresses.BuybackVault == (common.Address{}) {
		return nil
	}
	return bind.NewBoundContract(c.addresses.BuybackVault, vaultContractABI, c.caller, c.transactor, c.filterer)
}

func (c *Client) hook() *bind.BoundContract {
	if c == nil || c.addresses.MemeHook == (common.Address{}) {
		return nil
	}
	return bind.NewBoundContract(c.addresses.MemeHook, hookContractABI, c.caller, c.transactor, c.filterer)
}

func (c *Client) locker() *bind.BoundContract {
	if c == nil || c.addresses.LaunchLocker == (common.Address{}) {
		return nil
	}
	return bind.NewBoundContract(c.addresses.LaunchLocker, lockerContractABI, c.caller, c.transactor, c.filterer)
}

func (c *Client) call(ctx context.Context, contract *bind.BoundContract, opts *bind.CallOpts, method string, params ...interface{}) ([]interface{}, error) {
	if c == nil || isNilInterface(c.caller) {
		return nil, ErrNilBackend
	}
	if contract == nil {
		return nil, ErrZeroAddress
	}
	if ctx == nil {
		return nil, errors.New("nil context")
	}
	callOpts := normalizeCallOpts(ctx, opts)
	var out []interface{}
	if err := contract.Call(callOpts, &out, method, params...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) callStruct(ctx context.Context, contract *bind.BoundContract, opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	out, err := c.call(ctx, contract, opts, method, params...)
	if err != nil {
		return err
	}
	if len(out) != 1 {
		return errors.New("expected one tuple return value")
	}
	return mapABIValue(out[0], result)
}

func (c *Client) callAddress(ctx context.Context, contract *bind.BoundContract, opts *bind.CallOpts, method string, params ...interface{}) (common.Address, error) {
	out, err := c.call(ctx, contract, opts, method, params...)
	if err != nil {
		return common.Address{}, err
	}
	return out[0].(common.Address), nil
}

func (c *Client) callString(ctx context.Context, contract *bind.BoundContract, opts *bind.CallOpts, method string, params ...interface{}) (string, error) {
	out, err := c.call(ctx, contract, opts, method, params...)
	if err != nil {
		return "", err
	}
	return out[0].(string), nil
}

func (c *Client) callBig(ctx context.Context, contract *bind.BoundContract, opts *bind.CallOpts, method string, params ...interface{}) (*big.Int, error) {
	out, err := c.call(ctx, contract, opts, method, params...)
	if err != nil {
		return nil, err
	}
	return cloneBig(out[0].(*big.Int)), nil
}

func (c *Client) callBool(ctx context.Context, contract *bind.BoundContract, opts *bind.CallOpts, method string, params ...interface{}) (bool, error) {
	out, err := c.call(ctx, contract, opts, method, params...)
	if err != nil {
		return false, err
	}
	return out[0].(bool), nil
}

func normalizeCallOpts(ctx context.Context, opts *bind.CallOpts) *bind.CallOpts {
	if opts == nil {
		return &bind.CallOpts{Context: ctx}
	}
	cp := *opts
	cp.Context = ctx
	return &cp
}

func (c *Client) snapshotCallOpts(ctx context.Context, opts *bind.CallOpts) (*bind.CallOpts, error) {
	if ctx == nil {
		return nil, errors.New("nil context")
	}
	if c == nil || isNilInterface(c.caller) {
		return nil, ErrNilBackend
	}
	callOpts := normalizeCallOpts(ctx, opts)
	if callOpts.Pending || callOpts.BlockNumber != nil {
		return callOpts, nil
	}
	reader, ok := c.caller.(blockNumberReader)
	if !ok {
		return callOpts, nil
	}
	blockNumber, err := reader.BlockNumber(callOpts.Context)
	if err != nil {
		return nil, err
	}
	callOpts.BlockNumber = new(big.Int).SetUint64(blockNumber)
	return callOpts, nil
}

func parallelCalls(ctx context.Context, tasks ...func(context.Context) error) error {
	return parallelCallsLimit(ctx, len(tasks), tasks...)
}

func parallelCallsLimit(ctx context.Context, limit int, tasks ...func(context.Context) error) error {
	if len(tasks) == 0 {
		return nil
	}
	if ctx == nil {
		return errors.New("nil context")
	}
	if limit <= 0 {
		limit = 1
	}
	if limit > len(tasks) {
		limit = len(tasks)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan func(context.Context) error)
	errCh := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Add(limit)
	for range limit {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case task, ok := <-jobs:
					if !ok {
						return
					}
					if err := task(ctx); err != nil {
						select {
						case errCh <- err:
						default:
						}
						cancel()
						return
					}
				}
			}
		}()
	}

schedule:
	for _, task := range tasks {
		select {
		case <-ctx.Done():
			break schedule
		case jobs <- task:
		}
	}
	close(jobs)
	wg.Wait()
	select {
	case err := <-errCh:
		return err
	default:
	}
	return ctx.Err()
}

func taskCallOpts(opts *bind.CallOpts, ctx context.Context) *bind.CallOpts {
	cp := *opts
	cp.Context = ctx
	return &cp
}

func (c *Client) callAddressTask(contract *bind.BoundContract, opts *bind.CallOpts, out *common.Address, method string, params ...interface{}) func(context.Context) error {
	return func(ctx context.Context) error {
		value, err := c.callAddress(ctx, contract, taskCallOpts(opts, ctx), method, params...)
		if err == nil {
			*out = value
		}
		return err
	}
}

func (c *Client) callBigTask(contract *bind.BoundContract, opts *bind.CallOpts, out **big.Int, method string, params ...interface{}) func(context.Context) error {
	return func(ctx context.Context) error {
		value, err := c.callBig(ctx, contract, taskCallOpts(opts, ctx), method, params...)
		if err == nil {
			*out = value
		}
		return err
	}
}

func (c *Client) callBoolTask(contract *bind.BoundContract, opts *bind.CallOpts, out *bool, method string, params ...interface{}) func(context.Context) error {
	return func(ctx context.Context) error {
		value, err := c.callBool(ctx, contract, taskCallOpts(opts, ctx), method, params...)
		if err == nil {
			*out = value
		}
		return err
	}
}

func (c *Client) callStringTask(contract *bind.BoundContract, opts *bind.CallOpts, out *string, method string, params ...interface{}) func(context.Context) error {
	return func(ctx context.Context) error {
		value, err := c.callString(ctx, contract, taskCallOpts(opts, ctx), method, params...)
		if err == nil {
			*out = value
		}
		return err
	}
}

func (c *Client) transact(auth *bind.TransactOpts, address common.Address, payable bool, pack func() ([]byte, error)) (*types.Transaction, error) {
	if c == nil || isNilInterface(c.transactor) {
		return nil, ErrNilBackend
	}
	var err error
	if payable {
		err = validateAuth(auth)
	} else {
		err = validateNonPayableAuth(auth)
	}
	if err != nil {
		return nil, err
	}
	if err := validateAddress(address, "transaction destination"); err != nil {
		return nil, err
	}
	data, err := pack()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, abi.ABI{}, nil, c.transactor, nil).RawTransact(auth, data)
}

func (c *Client) factoryAddress() common.Address {
	if c == nil {
		return common.Address{}
	}
	return c.addresses.Factory
}

func (c *Client) hookAddress() common.Address {
	if c == nil {
		return common.Address{}
	}
	return c.addresses.MemeHook
}

func (c *Client) escrowAddress() common.Address {
	if c == nil {
		return common.Address{}
	}
	return c.addresses.FeeEscrow
}

func (c *Client) vaultAddress() common.Address {
	if c == nil {
		return common.Address{}
	}
	return c.addresses.BuybackVault
}

func (c *Client) routerAddress() common.Address {
	if c == nil {
		return common.Address{}
	}
	return c.addresses.LaunchAndBuy
}
