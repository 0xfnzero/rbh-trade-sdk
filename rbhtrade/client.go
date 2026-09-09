package rbhtrade

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type ChainIDReader interface {
	ChainID(context.Context) (*big.Int, error)
}

type DeploymentBackend interface {
	ChainIDReader
	CodeAt(context.Context, common.Address, *big.Int) ([]byte, error)
}

type Client struct {
	addresses AddressBook
}

func NewClient(ctx context.Context, backend ChainIDReader) (*Client, error) {
	if isNilInterface(backend) {
		return nil, fmt.Errorf("nil backend")
	}
	if ctx == nil {
		return nil, fmt.Errorf("nil context")
	}
	chainID, err := backend.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("read chain id: %w", err)
	}
	if chainID == nil || !chainID.IsInt64() || chainID.Int64() != ChainID {
		return nil, fmt.Errorf("got %v, want %d: %w", chainID, ChainID, ErrInvalidChainID)
	}
	return &Client{addresses: DefaultAddressBook()}, nil
}

func isNilInterface(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func (c *Client) Addresses() AddressBook {
	if c == nil {
		return AddressBook{}
	}
	return c.addresses
}

func NewClientChecked(ctx context.Context, backend DeploymentBackend) (*Client, error) {
	client, err := NewClient(ctx, backend)
	if err != nil {
		return nil, err
	}
	deployments := []struct {
		name    string
		address common.Address
	}{
		{"PoolManager", client.addresses.PoolManager},
		{"PositionManager", client.addresses.PositionManager},
		{"V4Quoter", client.addresses.V4Quoter},
		{"StateView", client.addresses.StateView},
		{"UniversalRouter", client.addresses.UniversalRouter},
		{"Permit2", client.addresses.Permit2},
		{"WETH", client.addresses.WETH},
		{"USDG", client.addresses.USDG},
		{"V3Factory", client.addresses.V3Factory},
		{"V3SwapRouter02", client.addresses.V3SwapRouter02},
		{"PonsFactory", client.addresses.PonsFactory},
		{"PonsLaunchAndBuy", client.addresses.PonsLaunchAndBuy},
		{"PonsMemeHook", client.addresses.PonsMemeHook},
		{"LongLauncher", client.addresses.LongLauncher},
		{"LongAirlock", client.addresses.LongAirlock},
		{"LongTokenFactory", client.addresses.LongTokenFactory},
		{"LongDopplerHook", client.addresses.LongDopplerHook},
		{"O1Factory", client.addresses.O1Factory},
		{"O1Hook", client.addresses.O1Hook},
		{"O1FeeEscrow", client.addresses.O1FeeEscrow},
		{"O1TokenDeployer", client.addresses.O1TokenDeployer},
		{"O1Registry", client.addresses.O1Registry},
		{"O1LaunchBuy", client.addresses.O1LaunchBuy},
		{"PoolsEntry", client.addresses.PoolsEntry},
		{"PoolsTokenFactory", client.addresses.PoolsTokenFactory},
		{"PoolsInstant", client.addresses.PoolsInstant},
		{"PAIRLaunchpad", client.addresses.PAIRLaunchpad},
		{"PAIRLocker", client.addresses.PAIRLocker},
		{"BagsFactory", client.addresses.BagsFactory},
		{"BagsHook", client.addresses.BagsHook},
		{"BagsVault", client.addresses.BagsVault},
		{"BagsLens", client.addresses.BagsLens},
	}
	errs := make([]error, len(deployments))
	var wg sync.WaitGroup
	for i, deployment := range deployments {
		wg.Add(1)
		go func() {
			defer wg.Done()
			code, codeErr := backend.CodeAt(ctx, deployment.address, nil)
			if codeErr != nil {
				errs[i] = fmt.Errorf("read %s code at %s: %w", deployment.name, deployment.address, codeErr)
				return
			}
			if len(code) == 0 {
				errs[i] = fmt.Errorf("no %s contract code at %s", deployment.name, deployment.address)
			}
		}()
	}
	wg.Wait()
	for _, deploymentErr := range errs {
		if deploymentErr != nil {
			return nil, deploymentErr
		}
	}
	return client, nil
}
