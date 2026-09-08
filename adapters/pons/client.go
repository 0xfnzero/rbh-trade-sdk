package pons

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Client struct {
	caller     bind.ContractCaller
	transactor bind.ContractTransactor
	filterer   bind.ContractFilterer
	addresses  Addresses
}

type Option func(*Client)

const (
	DefaultMaxLaunchConfigs        uint64 = 4096
	DefaultLaunchConfigConcurrency int    = 16
)

var ErrLaunchConfigLimit = errors.New("launch config enumeration limit exceeded")

// LaunchConfigQueryOptions bounds the memory and RPC concurrency used while
// enumerating launch configurations. Zero values select the safe defaults.
type LaunchConfigQueryOptions struct {
	MaxConfigs  uint64
	Concurrency int
}

func WithAddresses(addresses Addresses) Option {
	return func(c *Client) {
		c.addresses = addresses
	}
}

func NewClient(backend bind.ContractBackend, opts ...Option) *Client {
	client := &Client{
		caller:     backend,
		transactor: backend,
		filterer:   backend,
		addresses:  defaultPonsV2Addresses,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(client)
		}
	}
	return client
}

// NewClientChecked constructs a client and validates its backend and all
// addresses used directly by the SDK. Use it for fail-fast initialization.
func NewClientChecked(backend bind.ContractBackend, opts ...Option) (*Client, error) {
	if isNilInterface(backend) {
		return nil, ErrNilBackend
	}
	client := NewClient(backend, opts...)
	if err := client.addresses.Validate(); err != nil {
		return nil, err
	}
	return client, nil
}

func Dial(ctx context.Context, rpcURL string, opts ...Option) (*Client, *ethclient.Client, error) {
	if ctx == nil {
		return nil, nil, errors.New("nil context")
	}
	eth, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, nil, err
	}
	chainID, err := eth.ChainID(ctx)
	if err != nil {
		eth.Close()
		return nil, nil, err
	}
	if err := validateChainID(chainID); err != nil {
		eth.Close()
		return nil, nil, err
	}
	client, err := NewClientChecked(eth, opts...)
	if err != nil {
		eth.Close()
		return nil, nil, err
	}
	return client, eth, nil
}

func (c *Client) Addresses() Addresses {
	if c == nil {
		return Addresses{}
	}
	return c.addresses
}

func (c *Client) LaunchFee(ctx context.Context, opts *bind.CallOpts) (*big.Int, error) {
	return c.callBig(ctx, c.factory(), opts, "launchFee")
}

func (c *Client) LaunchEnabled(ctx context.Context, opts *bind.CallOpts) (bool, error) {
	return c.callBool(ctx, c.factory(), opts, "launchEnabled")
}

func (c *Client) CanLaunch(ctx context.Context, launcher common.Address, opts *bind.CallOpts) (bool, error) {
	return c.callBool(ctx, c.factory(), opts, "canLaunch", launcher)
}

func (c *Client) WhitelistedLauncher(ctx context.Context, launcher common.Address, opts *bind.CallOpts) (bool, error) {
	return c.callBool(ctx, c.factory(), opts, "whitelistedLaunchers", launcher)
}

func (c *Client) MaxCreatorTaxBps(ctx context.Context, opts *bind.CallOpts) (*big.Int, error) {
	return c.callBig(ctx, c.factory(), opts, "maxCreatorTaxBps")
}

func (c *Client) SnipeTaxStartBps(ctx context.Context, opts *bind.CallOpts) (*big.Int, error) {
	return c.callBig(ctx, c.factory(), opts, "snipeTaxStartBps")
}

func (c *Client) SnipeTaxSeconds(ctx context.Context, opts *bind.CallOpts) (*big.Int, error) {
	return c.callBig(ctx, c.factory(), opts, "snipeTaxSeconds")
}

func (c *Client) CreatorFeeRecipientTimelock(ctx context.Context, opts *bind.CallOpts) (*big.Int, error) {
	return c.callBig(ctx, c.factory(), opts, "CREATOR_FEE_RECIPIENT_TIMELOCK")
}

func (c *Client) CreatorFeeRecipientExecutionWindow(ctx context.Context, opts *bind.CallOpts) (*big.Int, error) {
	return c.callBig(ctx, c.factory(), opts, "CREATOR_FEE_RECIPIENT_EXECUTION_WINDOW")
}

func (c *Client) GraduationRescueDelay(ctx context.Context, opts *bind.CallOpts) (*big.Int, error) {
	return c.callBig(ctx, c.factory(), opts, "GRADUATION_RESCUE_DELAY")
}

func (c *Client) FactoryOwner(ctx context.Context, opts *bind.CallOpts) (common.Address, error) {
	return c.callAddress(ctx, c.factory(), opts, "owner")
}

func (c *Client) FactoryPendingOwner(ctx context.Context, opts *bind.CallOpts) (common.Address, error) {
	return c.callAddress(ctx, c.factory(), opts, "pendingOwner")
}

func (c *Client) LaunchConfigCount(ctx context.Context, opts *bind.CallOpts) (*big.Int, error) {
	return c.callBig(ctx, c.factory(), opts, "launchConfigCount")
}

func (c *Client) GetLaunchConfig(ctx context.Context, id *big.Int, opts *bind.CallOpts) (LaunchConfig, error) {
	if err := validateUint256(id, "launch config ID"); err != nil {
		return LaunchConfig{}, err
	}
	var encoded launchConfigABI
	if err := c.callStruct(ctx, c.factory(), opts, &encoded, "getLaunchConfig", id); err != nil {
		return LaunchConfig{}, err
	}
	poolFee, err := uint24ToUint32(encoded.PoolFee, "pool fee")
	if err != nil {
		return LaunchConfig{}, err
	}
	tickSpacing, err := int24ToInt32(encoded.TickSpacing, "tick spacing")
	if err != nil {
		return LaunchConfig{}, err
	}
	return LaunchConfig{
		Supply:              cloneBig(encoded.Supply),
		CurveFeeBps:         cloneBig(encoded.CurveFeeBps),
		PhantomQuote:        cloneBig(encoded.PhantomQuote),
		GraduationThreshold: cloneBig(encoded.GraduationThreshold),
		PoolFee:             poolFee,
		TickSpacing:         tickSpacing,
		Enabled:             encoded.Enabled,
	}, nil
}

func (c *Client) OpenLaunchConfigs(ctx context.Context, opts *bind.CallOpts) ([]OpenLaunchConfig, error) {
	return c.OpenLaunchConfigsWithOptions(ctx, LaunchConfigQueryOptions{}, opts)
}

func (c *Client) OpenLaunchConfigsWithOptions(ctx context.Context, query LaunchConfigQueryOptions, opts *bind.CallOpts) ([]OpenLaunchConfig, error) {
	callOpts, err := c.snapshotCallOpts(ctx, opts)
	if err != nil {
		return nil, err
	}
	count, err := c.LaunchConfigCount(ctx, callOpts)
	if err != nil {
		return nil, err
	}
	maxConfigs := query.MaxConfigs
	if maxConfigs == 0 {
		maxConfigs = DefaultMaxLaunchConfigs
	}
	n, err := launchConfigEnumerationSize(count, maxConfigs)
	if err != nil {
		return nil, err
	}
	all := make([]LaunchConfig, n)
	tasks := make([]func(context.Context) error, n)
	nextID := new(big.Int)
	one := big.NewInt(1)
	for i := 0; i < n; i++ {
		i := i
		id := new(big.Int).Set(nextID)
		nextID.Add(nextID, one)
		tasks[i] = func(taskCtx context.Context) error {
			taskOpts := *callOpts
			taskOpts.Context = taskCtx
			config, err := c.GetLaunchConfig(taskCtx, id, &taskOpts)
			if err == nil {
				all[i] = config
			}
			return err
		}
	}
	concurrent := query.Concurrency
	if concurrent <= 0 {
		concurrent = DefaultLaunchConfigConcurrency
	}
	if err := parallelCallsLimit(callOpts.Context, concurrent, tasks...); err != nil {
		return nil, err
	}

	configs := make([]OpenLaunchConfig, 0, n)
	for i, config := range all {
		if config.Enabled {
			configs = append(configs, OpenLaunchConfig{ID: big.NewInt(int64(i)), LaunchConfig: config})
		}
	}
	return configs, nil
}

func launchConfigEnumerationSize(count *big.Int, maxConfigs uint64) (int, error) {
	if count == nil || count.Sign() < 0 || !count.IsUint64() {
		return 0, errors.New("launch config count is too large for local enumeration")
	}
	n := count.Uint64()
	if n > maxConfigs {
		return 0, fmt.Errorf("%w: count %d exceeds maximum %d", ErrLaunchConfigLimit, n, maxConfigs)
	}
	if n > uint64(^uint(0)>>1) {
		return 0, errors.New("launch config count exceeds local integer capacity")
	}
	return int(n), nil
}

func (c *Client) PreviewLaunchEconomics(ctx context.Context, launchConfigID *big.Int, pairToken common.Address, opts *bind.CallOpts) (common.Hash, error) {
	if err := validateUint256(launchConfigID, "launch config ID"); err != nil {
		return common.Hash{}, err
	}
	out, err := c.call(ctx, c.factory(), opts, "previewLaunchEconomics", launchConfigID, pairToken)
	if err != nil {
		return common.Hash{}, err
	}
	return common.Hash(out[0].([32]byte)), nil
}

func (c *Client) ApprovedPairToken(ctx context.Context, pairToken common.Address, opts *bind.CallOpts) (bool, error) {
	return c.callBool(ctx, c.factory(), opts, "approvedPairTokens", pairToken)
}

func (c *Client) PairTokenEconomics(ctx context.Context, pairToken common.Address, opts *bind.CallOpts) (PairTokenEconomics, error) {
	out, err := c.call(ctx, c.factory(), opts, "pairTokenEconomics", pairToken)
	if err != nil {
		return PairTokenEconomics{}, err
	}
	return PairTokenEconomics{
		PhantomQuote:        cloneBig(out[0].(*big.Int)),
		GraduationThreshold: cloneBig(out[1].(*big.Int)),
		Decimals:            out[2].(uint8),
	}, nil
}

func (c *Client) PendingCreatorFeeRecipient(ctx context.Context, token common.Address, opts *bind.CallOpts) (PendingCreatorFeeRecipient, error) {
	out, err := c.call(ctx, c.factory(), opts, "pendingCreatorFeeRecipient", token)
	if err != nil {
		return PendingCreatorFeeRecipient{}, err
	}
	return PendingCreatorFeeRecipient{
		NewRecipient: out[0].(common.Address),
		EffectiveAt:  cloneBig(out[1].(*big.Int)),
		ExpiresAt:    cloneBig(out[2].(*big.Int)),
	}, nil
}

func (c *Client) GetLaunchedToken(ctx context.Context, token common.Address, opts *bind.CallOpts) (LaunchedToken, error) {
	var encoded launchedTokenABI
	if err := c.callStruct(ctx, c.factory(), opts, &encoded, "getLaunchedToken", token); err != nil {
		return LaunchedToken{}, err
	}
	poolFee, err := uint24ToUint32(encoded.PoolFee, "pool fee")
	if err != nil {
		return LaunchedToken{}, err
	}
	tickSpacing, err := int24ToInt32(encoded.TickSpacing, "tick spacing")
	if err != nil {
		return LaunchedToken{}, err
	}
	return LaunchedToken{
		Token:               encoded.Token,
		Curve:               encoded.Curve,
		Deployer:            encoded.Deployer,
		CreatorFeeRecipient: encoded.CreatorFeeRecipient,
		PairToken:           encoded.PairToken,
		GraduationThreshold: cloneBig(encoded.GraduationThreshold),
		PoolFee:             poolFee,
		TickSpacing:         tickSpacing,
		CreatorTaxBps:       encoded.CreatorTaxBps,
		BuybackEnabled:      encoded.BuybackEnabled,
		Phase:               encoded.Phase,
		SweptQuote:          cloneBig(encoded.SweptQuote),
		SweptTokens:         cloneBig(encoded.SweptTokens),
		SweptAt:             cloneBig(encoded.SweptAt),
		Exists:              encoded.Exists,
	}, nil
}

func (c *Client) GetLaunchFeePolicy(ctx context.Context, token common.Address, opts *bind.CallOpts) (FeePolicy, error) {
	var policy FeePolicy
	err := c.callStruct(ctx, c.factory(), opts, &policy, "getLaunchFeePolicy", token)
	return policy, err
}

func (c *Client) CurrentFeePolicy(ctx context.Context, opts *bind.CallOpts) (FeePolicy, error) {
	var policy FeePolicy
	err := c.callStruct(ctx, c.hook(), opts, &policy, "currentFeePolicy")
	return policy, err
}

func (c *Client) HookLaunch(ctx context.Context, poolID common.Hash, opts *bind.CallOpts) (HookLaunch, error) {
	out, err := c.call(ctx, c.hook(), opts, "launches", [32]byte(poolID))
	if err != nil {
		return HookLaunch{}, err
	}
	return HookLaunch{
		Registered:                out[0].(bool),
		MemecoinIsCurrency0:       out[1].(bool),
		Memecoin:                  out[2].(common.Address),
		QuoteToken:                out[3].(common.Address),
		Creator:                   out[4].(common.Address),
		BuybackCreatorRecipient:   out[5].(common.Address),
		ProtocolFeeRecipient:      out[6].(common.Address),
		CreatorTaxBps:             out[7].(uint16),
		ProtocolFeeShareBps:       out[8].(uint16),
		BuybackBurnBps:            out[9].(uint16),
		HookFeeBps:                out[10].(uint16),
		MaxInternalPriceImpactBps: out[11].(uint16),
		BuybackEnabled:            out[12].(bool),
	}, nil
}

func (c *Client) GetReserves(ctx context.Context, curve common.Address, opts *bind.CallOpts) (CurveReserves, error) {
	out, err := c.call(ctx, c.curve(curve), opts, "getReserves")
	if err != nil {
		return CurveReserves{}, err
	}
	return CurveReserves{QuoteReserve: cloneBig(out[0].(*big.Int)), TokenReserve: cloneBig(out[1].(*big.Int))}, nil
}

// CurrentSnipeTaxBps returns the recipient-specific tax used by the local
// QuoteBuyFromState calculation. Callers can refresh it off the trading path.
func (c *Client) CurrentSnipeTaxBps(ctx context.Context, curve, recipient common.Address, opts *bind.CallOpts) (*big.Int, error) {
	out, err := c.call(ctx, c.curve(curve), opts, "currentSnipeTaxBps", recipient)
	if err != nil {
		return nil, err
	}
	return cloneBig(out[0].(*big.Int)), nil
}

func (c *Client) CurveState(ctx context.Context, curve common.Address, opts *bind.CallOpts) (CurveState, error) {
	callOpts, err := c.snapshotCallOpts(ctx, opts)
	if err != nil {
		return CurveState{}, err
	}
	contract := c.curve(curve)
	var token, pairToken, deployer, factory common.Address
	var reserves CurveReserves
	var phantomQuote, realQuoteReserve, threshold, sellable, reservedTokens *big.Int
	var trackedQuote, trackedTokens, quoteFeeBalance, buybackQuoteBalance, creatorTaxBalance *big.Int
	var feeBps, creatorTaxBps *big.Int
	var ready, graduated, buybackEnabled bool
	err = parallelCalls(callOpts.Context,
		c.callAddressTask(contract, callOpts, &token, "token"),
		c.callAddressTask(contract, callOpts, &pairToken, "pairToken"),
		c.callAddressTask(contract, callOpts, &deployer, "deployer"),
		c.callAddressTask(contract, callOpts, &factory, "factory"),
		func(taskCtx context.Context) error {
			out, err := c.call(taskCtx, contract, taskCallOpts(callOpts, taskCtx), "getReserves")
			if err == nil {
				reserves = CurveReserves{QuoteReserve: cloneBig(out[0].(*big.Int)), TokenReserve: cloneBig(out[1].(*big.Int))}
			}
			return err
		},
		c.callBigTask(contract, callOpts, &phantomQuote, "phantomQuote"),
		c.callBigTask(contract, callOpts, &realQuoteReserve, "realQuoteReserve"),
		c.callBigTask(contract, callOpts, &threshold, "graduationThreshold"),
		c.callBigTask(contract, callOpts, &sellable, "sellableTokens"),
		c.callBigTask(contract, callOpts, &reservedTokens, "reservedTokens"),
		c.callBigTask(contract, callOpts, &trackedQuote, "trackedQuote"),
		c.callBigTask(contract, callOpts, &trackedTokens, "trackedTokens"),
		c.callBigTask(contract, callOpts, &quoteFeeBalance, "quoteFeeBalance"),
		c.callBigTask(contract, callOpts, &buybackQuoteBalance, "buybackQuoteBalance"),
		c.callBigTask(contract, callOpts, &creatorTaxBalance, "creatorTaxBalance"),
		c.callBoolTask(contract, callOpts, &ready, "readyToGraduate"),
		c.callBoolTask(contract, callOpts, &graduated, "graduated"),
		c.callBigTask(contract, callOpts, &feeBps, "feeBps"),
		c.callBigTask(contract, callOpts, &creatorTaxBps, "creatorTaxBps"),
		c.callBoolTask(contract, callOpts, &buybackEnabled, "buybackEnabled"),
	)
	if err != nil {
		return CurveState{}, err
	}
	return CurveState{
		Curve:               curve,
		Token:               token,
		PairToken:           pairToken,
		Deployer:            deployer,
		Factory:             factory,
		Reserves:            reserves,
		PhantomQuote:        phantomQuote,
		RealQuoteReserve:    realQuoteReserve,
		GraduationThreshold: threshold,
		SellableTokens:      sellable,
		ReservedTokens:      reservedTokens,
		TrackedQuote:        trackedQuote,
		TrackedTokens:       trackedTokens,
		QuoteFeeBalance:     quoteFeeBalance,
		BuybackQuoteBalance: buybackQuoteBalance,
		CreatorTaxBalance:   creatorTaxBalance,
		ReadyToGraduate:     ready,
		Graduated:           graduated,
		FeeBps:              feeBps,
		CreatorTaxBps:       creatorTaxBps,
		BuybackEnabled:      buybackEnabled,
	}, nil
}

func (c *Client) IsNativeQuote(ctx context.Context, curve common.Address, opts *bind.CallOpts) (bool, error) {
	return c.callBool(ctx, c.curve(curve), opts, "isNativeQuote")
}

func (c *Client) TokenInfo(ctx context.Context, token common.Address, opts *bind.CallOpts) (TokenInfo, error) {
	out, err := c.call(ctx, c.token(token), opts, "getTokenInfo")
	if err != nil {
		return TokenInfo{}, err
	}
	socials, err := convert[Socials](out[3])
	if err != nil {
		return TokenInfo{}, err
	}
	return TokenInfo{Deployer: out[0].(common.Address), Logo: out[1].(string), Description: out[2].(string), Socials: socials}, nil
}

func (c *Client) TokenMetadata(ctx context.Context, token common.Address, opts *bind.CallOpts) (TokenMetadata, error) {
	callOpts, err := c.snapshotCallOpts(ctx, opts)
	if err != nil {
		return TokenMetadata{}, err
	}
	contract := c.token(token)
	var name, symbol string
	var decimals uint8
	var totalSupply *big.Int
	var launchFactory, curve common.Address
	var info TokenInfo
	err = parallelCalls(callOpts.Context,
		c.callStringTask(contract, callOpts, &name, "name"),
		c.callStringTask(contract, callOpts, &symbol, "symbol"),
		func(taskCtx context.Context) error {
			out, err := c.call(taskCtx, contract, taskCallOpts(callOpts, taskCtx), "decimals")
			if err == nil {
				decimals = out[0].(uint8)
			}
			return err
		},
		c.callBigTask(contract, callOpts, &totalSupply, "totalSupply"),
		c.callAddressTask(contract, callOpts, &launchFactory, "launchFactory"),
		c.callAddressTask(contract, callOpts, &curve, "curve"),
		func(taskCtx context.Context) error {
			out, err := c.call(taskCtx, contract, taskCallOpts(callOpts, taskCtx), "getTokenInfo")
			if err != nil {
				return err
			}
			socials, err := convert[Socials](out[3])
			if err == nil {
				info = TokenInfo{Deployer: out[0].(common.Address), Logo: out[1].(string), Description: out[2].(string), Socials: socials}
			}
			return err
		},
	)
	if err != nil {
		return TokenMetadata{}, err
	}
	return TokenMetadata{
		Name:          name,
		Symbol:        symbol,
		Decimals:      decimals,
		TotalSupply:   totalSupply,
		Deployer:      info.Deployer,
		LaunchFactory: launchFactory,
		Curve:         curve,
		Logo:          info.Logo,
		Description:   info.Description,
		Socials:       info.Socials,
	}, nil
}

func (c *Client) TokenAllowance(ctx context.Context, token common.Address, owner common.Address, spender common.Address, opts *bind.CallOpts) (*big.Int, error) {
	return c.callBig(ctx, c.token(token), opts, "allowance", owner, spender)
}

func (c *Client) EscrowNativeBalance(ctx context.Context, recipient common.Address, opts *bind.CallOpts) (*big.Int, error) {
	return c.callBig(ctx, c.escrow(), opts, "balanceOf", recipient)
}

func (c *Client) EscrowTokenBalance(ctx context.Context, recipient common.Address, token common.Address, opts *bind.CallOpts) (*big.Int, error) {
	return c.callBig(ctx, c.escrow(), opts, "balanceOfToken", recipient, token)
}

func (c *Client) VestState(ctx context.Context, token common.Address, opts *bind.CallOpts) (VestState, error) {
	callOpts, err := c.snapshotCallOpts(ctx, opts)
	if err != nil {
		return VestState{}, err
	}
	contract := c.vault()
	var totalLocked, totalReleased, vested, releasable, start, duration *big.Int
	var terms VestingTerms
	err = parallelCalls(callOpts.Context,
		c.callBigTask(contract, callOpts, &totalLocked, "totalLocked", token),
		c.callBigTask(contract, callOpts, &totalReleased, "totalReleased", token),
		c.callBigTask(contract, callOpts, &vested, "vestedAmount", token),
		c.callBigTask(contract, callOpts, &releasable, "releasable", token),
		c.callBigTask(contract, callOpts, &start, "vestingStart", token),
		c.callBigTask(contract, callOpts, &duration, "VESTING_DURATION"),
		func(taskCtx context.Context) error {
			out, err := c.call(taskCtx, contract, taskCallOpts(callOpts, taskCtx), "vestingTerms", token)
			if err == nil {
				terms = VestingTerms{
					CreatorRecipient:    out[0].(common.Address),
					ProtocolRecipient:   out[1].(common.Address),
					ProtocolFeeShareBps: out[2].(uint16),
				}
			}
			return err
		},
	)
	if err != nil {
		return VestState{}, err
	}
	return VestState{TotalLocked: totalLocked, TotalReleased: totalReleased, VestedAmount: vested, Releasable: releasable, VestingStart: start, VestingDuration: duration, Terms: terms}, nil
}

func (c *Client) VestingTerms(ctx context.Context, token common.Address, opts *bind.CallOpts) (VestingTerms, error) {
	out, err := c.call(ctx, c.vault(), opts, "vestingTerms", token)
	if err != nil {
		return VestingTerms{}, err
	}
	return VestingTerms{
		CreatorRecipient:    out[0].(common.Address),
		ProtocolRecipient:   out[1].(common.Address),
		ProtocolFeeShareBps: out[2].(uint16),
	}, nil
}

func (c *Client) PendingFees(ctx context.Context, poolID common.Hash, currency common.Address, opts *bind.CallOpts) (*big.Int, error) {
	return c.callBig(ctx, c.hook(), opts, "pendingFees", [32]byte(poolID), currency)
}

func (c *Client) PendingCreatorTax(ctx context.Context, poolID common.Hash, currency common.Address, opts *bind.CallOpts) (*big.Int, error) {
	return c.callBig(ctx, c.hook(), opts, "pendingCreatorTax", [32]byte(poolID), currency)
}

func (c *Client) PendingBuyback(ctx context.Context, poolID common.Hash, currency common.Address, opts *bind.CallOpts) (*big.Int, error) {
	return c.callBig(ctx, c.hook(), opts, "pendingBuyback", [32]byte(poolID), currency)
}

func (c *Client) LockerIsLocked(ctx context.Context, token common.Address, opts *bind.CallOpts) (bool, error) {
	return c.callBool(ctx, c.locker(), opts, "isLocked", token)
}

func (c *Client) LaunchTokenSimple(auth *bind.TransactOpts, params TokenParams, launchConfigID *big.Int, pairToken common.Address) (*types.Transaction, error) {
	return c.transact(auth, c.factoryAddress(), true, func() ([]byte, error) {
		return PackLaunchTokenSimple(params, launchConfigID, pairToken)
	})
}

func (c *Client) LaunchToken(auth *bind.TransactOpts, params TokenParams, launchConfigID *big.Int, pairToken common.Address, snipeTaxExemptions []common.Address) (*types.Transaction, error) {
	return c.transact(auth, c.factoryAddress(), true, func() ([]byte, error) {
		return PackLaunchToken(params, launchConfigID, pairToken, snipeTaxExemptions)
	})
}

func (c *Client) LaunchTokenFor(auth *bind.TransactOpts, params TokenParams, launchConfigID *big.Int, pairToken common.Address, originalDeployer common.Address, snipeTaxExemptions []common.Address) (*types.Transaction, error) {
	return c.transact(auth, c.factoryAddress(), true, func() ([]byte, error) {
		return PackLaunchTokenFor(params, launchConfigID, pairToken, originalDeployer, snipeTaxExemptions)
	})
}

func (c *Client) LaunchAndBuy(auth *bind.TransactOpts, params TokenParams, launchConfigID *big.Int, pairToken common.Address, quoteIn *big.Int, minTokensOut *big.Int, recipient common.Address, snipeTaxExemptions []common.Address) (*types.Transaction, error) {
	return c.transact(auth, c.routerAddress(), true, func() ([]byte, error) {
		return PackLaunchAndBuy(params, launchConfigID, pairToken, quoteIn, minTokensOut, recipient, snipeTaxExemptions)
	})
}

func (c *Client) Buy(auth *bind.TransactOpts, curve common.Address, quoteIn *big.Int, minTokensOut *big.Int, recipient common.Address) (*types.Transaction, error) {
	return c.transact(auth, curve, true, func() ([]byte, error) { return PackBuy(quoteIn, minTokensOut, recipient) })
}

func (c *Client) Sell(auth *bind.TransactOpts, curve common.Address, tokensIn *big.Int, minQuoteOut *big.Int, recipient common.Address) (*types.Transaction, error) {
	return c.transact(auth, curve, false, func() ([]byte, error) { return PackSell(tokensIn, minQuoteOut, recipient) })
}

func (c *Client) SweepCurveFees(auth *bind.TransactOpts, curve common.Address, minBuybackTokensOut *big.Int) (*types.Transaction, error) {
	return c.transact(auth, curve, false, func() ([]byte, error) { return PackSweepCurveFees(minBuybackTokensOut) })
}

func (c *Client) Graduate(auth *bind.TransactOpts, token common.Address) (*types.Transaction, error) {
	return c.transact(auth, c.factoryAddress(), false, func() ([]byte, error) { return PackGraduate(token) })
}

func (c *Client) CreateGraduatedPool(auth *bind.TransactOpts, token common.Address) (*types.Transaction, error) {
	return c.transact(auth, c.factoryAddress(), false, func() ([]byte, error) { return PackCreateGraduatedPool(token) })
}

func (c *Client) SweepPoolFees(auth *bind.TransactOpts, poolID common.Hash, minConversionQuoteOut *big.Int, minBuybackTokensOut *big.Int) (*types.Transaction, error) {
	return c.transact(auth, c.hookAddress(), false, func() ([]byte, error) {
		return PackSweepPoolFees(poolID, minConversionQuoteOut, minBuybackTokensOut)
	})
}

func (c *Client) TransferCreatorFeeRecipient(auth *bind.TransactOpts, token common.Address, newRecipient common.Address) (*types.Transaction, error) {
	return c.transact(auth, c.factoryAddress(), false, func() ([]byte, error) {
		return PackTransferCreatorFeeRecipient(token, newRecipient)
	})
}

func (c *Client) SetBuybackEnabled(auth *bind.TransactOpts, token common.Address, enabled bool) (*types.Transaction, error) {
	return c.transact(auth, c.factoryAddress(), false, func() ([]byte, error) { return PackSetBuybackEnabled(token, enabled) })
}

func (c *Client) Approve(auth *bind.TransactOpts, token common.Address, spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return c.transact(auth, token, false, func() ([]byte, error) { return PackApprove(spender, amount) })
}

func (c *Client) ClaimNative(auth *bind.TransactOpts) (*types.Transaction, error) {
	return c.transact(auth, c.escrowAddress(), false, PackClaimNative)
}

func (c *Client) ClaimNativeAmount(auth *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return c.transact(auth, c.escrowAddress(), false, func() ([]byte, error) { return PackClaimNativeAmount(amount) })
}

func (c *Client) ClaimToken(auth *bind.TransactOpts, token common.Address) (*types.Transaction, error) {
	return c.transact(auth, c.escrowAddress(), false, func() ([]byte, error) { return PackClaimToken(token) })
}

func (c *Client) ClaimTokenAmount(auth *bind.TransactOpts, token common.Address, amount *big.Int) (*types.Transaction, error) {
	return c.transact(auth, c.escrowAddress(), false, func() ([]byte, error) { return PackClaimTokenAmount(token, amount) })
}

func (c *Client) ReleaseVested(auth *bind.TransactOpts, token common.Address) (*types.Transaction, error) {
	return c.transact(auth, c.vaultAddress(), false, func() ([]byte, error) { return PackReleaseVested(token) })
}

func (c *Client) ReleaseVest(auth *bind.TransactOpts, token common.Address) (*types.Transaction, error) {
	return c.ReleaseVested(auth, token)
}
