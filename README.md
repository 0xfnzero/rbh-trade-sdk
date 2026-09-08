<div align="center">
  <h1>RBH Trade SDK for Go</h1>
  <h3><em>Low-latency Go SDK for Robinhood Chain launchpads and Uniswap v4 trading</em></h3>
</div>

<p align="center">
  <strong>Typed calldata builders for Pons V2, Long, o1, Pools.trade, PAIR, Bags V2, Permit2, and the Robinhood Chain Universal Router.</strong>
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/0xfnzero/rbh-trade-sdk"><img src="https://pkg.go.dev/badge/github.com/0xfnzero/rbh-trade-sdk.svg" alt="Go Reference"></a>
  <a href="https://github.com/0xfnzero/rbh-trade-sdk/releases/latest"><img src="https://img.shields.io/github/v/release/0xfnzero/rbh-trade-sdk" alt="Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT License"></a>
</p>

<p align="center">
  <a href="./README_CN.md">中文</a> |
  <a href="./README.md">English</a> |
  <a href="https://fnzero.dev/">Website</a> |
  <a href="https://t.me/fnzero_group">Telegram</a> |
  <a href="https://discord.gg/vuazbGkqQE">Discord</a>
</p>

## SDKs

| SDK | Module |
|-----|--------|
| Trade | [`github.com/0xfnzero/rbh-trade-sdk`](https://github.com/0xfnzero/rbh-trade-sdk) |
| Parser | [`github.com/0xfnzero/rbh-parser-sdk`](https://github.com/0xfnzero/rbh-parser-sdk) |

## Coverage

| Protocol | Launch lifecycle | Trading path |
|----------|------------------|--------------|
| Pons V2 | Bonding curve, then Uniswap v4 | Native Pons client and curve quotes before graduation; shared v4 builder after graduation |
| Long | Direct Doppler/Airlock Uniswap v4 | Shared v4 builder using the exact `Initialize` PoolKey |
| o1 | Permanent Uniswap v4 liquidity | Shared v4 builder with optional hook data |
| Pools.trade | Instant v4 or crowdsale, then v4 | Pools launch builders and shared v4 builder |
| PAIR | One to five permanent v4 pools | PAIR multi-market launch and shared v4 builder |
| Bags V2 | Bonding curve, then Uniswap v4 | Bags curve builders and shared v4 builder |

`SupportedProtocols()` exposes machine-readable `ProtocolCapabilities` for
launch, curve, v4, deterministic-address, and verified local-pricing support.
Applications should use those flags to fail closed instead of treating a
calldata builder as proof that pre-execution pricing is available.

## Features

- Robinhood Chain `4663` deployment catalog and optional bytecode validation.
- Robinhood-specific Universal Router v4 exact-input encoding, including the required `minHopPriceX36` field.
- ERC-20 and Permit2 approval builders.
- V4 Quoter and StateView request/response helpers.
- Local V4 exact-input quotes from event-derived tick/liquidity state, using the audited Uniswap core math and optional output-hook fee cuts.
- Verified o1 LaunchHook quote-fee and linear anti-snipe decay math via `BuildO1HookPoolConfig`, `DecodeO1HookPoolConfig`, `O1HookFeeBPS`, and `QuoteO1V4ExactInput`; callers must prewarm the frozen pool schedule and supply the target block timestamp.
- Verified Bags V4 hook pricing via `QuoteBagsV4ExactInput`: the hook overrides the core LP fee to zero and charges 2% on the WETH leg.
- Launch builders for Long, o1, Pools.trade, PAIR, and Bags V2.
- Complete Pons V2 ABI, typed read client, curve quote math, calldata builders, and transaction wrappers.
- Deterministic Pons CREATE2 launch prediction pinned to Sourcify exact-match deployer bytecode, including economics commitment and runtime code-hash verification.
- No hidden RPC calls in calldata builders and no private-key custody.

## Installation

### Direct Clone

Clone the released source into your project directory:

```bash
cd your_project_root_directory
git clone --branch v0.1.0 --depth 1 https://github.com/0xfnzero/rbh-trade-sdk
```

Add the local module to your application's `go.mod`:

```go
require github.com/0xfnzero/rbh-trade-sdk v0.1.0

replace github.com/0xfnzero/rbh-trade-sdk => ./rbh-trade-sdk
```

Then run:

```bash
go mod tidy
```

### Go Modules

```bash
go get github.com/0xfnzero/rbh-trade-sdk@v0.1.0
```

## Build a v4 Swap

Always obtain the complete `PoolKey` from the PoolManager `Initialize` event. Do not infer Long's hook from `LaunchCreated.poolOrHook`.

```go
package main

import (
    "math/big"
    "time"

    sdk "github.com/0xfnzero/rbh-trade-sdk/rbhtrade"
    "github.com/ethereum/go-ethereum/common"
)

func main() {
    token := common.HexToAddress("0xYourToken")
    quote := common.HexToAddress("0xYourQuoteToken")

    call, err := sdk.BuildV4ExactInputSingle(sdk.ExactInputRequest{
        PoolKey: sdk.PoolKey{
            Currency0:   quote, // currencies must be address-sorted
            Currency1:   token,
            Fee:         0x800000,
            TickSpacing: 8,
            Hooks:       common.HexToAddress("0xPoolHookFromInitialize"),
        },
        CurrencyIn:       quote,
        CurrencyOut:      token,
        AmountIn:         big.NewInt(1_000_000),
        AmountOutMinimum: big.NewInt(900_000),
        Deadline:         uint64(time.Now().Add(30 * time.Second).Unix()),
    })
    if err != nil {
        panic(err)
    }

    // Sign and send call.To, call.Data, and call.Value with your own stack.
    _ = call
}
```

For an ERC-20 input, authorize Permit2 and then the Universal Router:

```go
approvals, err := sdk.BuildUniversalRouterApprovals(
    quote,
    big.NewInt(1_000_000),
    uint64(time.Now().Add(24*time.Hour).Unix()),
)
```

Native input uses the zero address as `CurrencyIn`; the swap call automatically sets `Value = AmountIn` and needs no Permit2 approval.

## Quote and Pool State

```go
quoteCall, err := sdk.BuildV4QuoteExactInputSingle(sdk.QuoteExactInputRequest{
    PoolKey: key, CurrencyIn: quoteToken, AmountIn: amountIn, HookData: hookData,
})
// eth_call quoteCall, then:
result, err := sdk.DecodeV4QuoteExactInputSingle(returnData)

poolID, err := sdk.PoolID(key)
slotCall, err := sdk.BuildStateViewGetSlot0(poolID)
slot0, err := sdk.DecodeStateViewSlot0(slotReturnData)
```

## Long Launches

`BuildLongCreate` implements the verified `LongLauncher.create` selector `0x882db707`. The caller supplies the official Doppler/Airlock factory payloads; the SDK rejects a token factory other than Long's trusted deployment.

```go
call, err := sdk.BuildLongCreate(sdk.LongCreateParams{
    InitialSupply:     initialSupply,
    NumTokensToSell:   tokensToSell,
    Numeraire:         stockOrQuoteToken,
    TokenFactory:      sdk.DefaultAddressBook().LongTokenFactory,
    TokenFactoryData:  tokenFactoryData,
    GovernanceFactory: governanceFactory,
    PoolInitializer:   poolInitializer,
    PoolInitializerData: poolInitializerData,
    LiquidityMigrator: liquidityMigrator,
    Salt:              salt,
})
```

Use the sibling parser to pair `LaunchCreated` with the same transaction's PoolManager `Initialize`; only that event contains the actual PoolKey.

## Other Launchpads

| Protocol | Builder | Value and funding rule |
|----------|---------|------------------------|
| o1 | `BuildO1Launch`, `BuildO1LaunchAndBuy` | Pass the current launch fee in `value`; native buy funding requires `value >= AmountIn` |
| Pools.trade | `BuildPoolsCreateToken`, `BuildPoolsDistributeToken`, `BuildPoolsDistributeWithNative`, `BuildPoolsMulticall` | Pass the exact fee/funding value quoted by the protocol; empty multicalls are rejected |
| PAIR | `BuildPAIRLaunch` | Allocation weights must total 10,000 BPS; use `PAIRNoDeveloperBuy` with zero buy amounts to disable the developer buy |
| Bags V2 | `BuildBagsCreate`, `BuildBagsCreateAndBuy`, `BuildBagsBuy`, `BuildBagsSell` | Curve buys are funded by `value`; curve sells require token approval outside this SDK |

Read fee, configuration, quote, and deadline inputs from the target protocol immediately before building the call. Builders validate ABI bounds and structural invariants, but they do not fetch mutable on-chain configuration.

## Pons V2

```go
import pons "github.com/0xfnzero/rbh-trade-sdk/adapters/pons"

client := pons.NewClient(backend)
state, err := client.CurveState(ctx, curve, nil)
quote, err := pons.QuoteBuyFromState(
    state.Reserves,
    state.SellableTokens,
    quoteIn,
    state.FeeBps,
    state.CreatorTaxBps,
    snipeTaxBps,
)
minTokensOut, err := pons.MinTokensOutForBuy(quoteIn, quote, slippageBps)
buyCall, err := pons.BuildBuy(curve, quoteIn, minTokensOut, recipient, true)
sellCall, err := pons.BuildSell(curve, tokensIn, minQuoteOut, recipient)
```

The Pons package is implemented entirely inside this module. No separate Pons SDK dependency is required.

## Deployment Validation

```go
backend, err := ethclient.DialContext(ctx, rpcURL)
client, err := sdk.NewClientChecked(ctx, backend)
```

`NewClientChecked` verifies chain id `4663` and bytecode at every fixed contract address used directly by a builder. This belongs at startup, not in the trading hot path.

Run the live checks with:

```bash
ROBINHOOD_RPC_URL=https://rpc.mainnet.chain.robinhood.com go test ./rbhtrade -run TestRobinhoodDeployment
```

## Security Notes

1. Simulate or estimate every value-bearing call against the intended block before signing.
2. Set a non-zero `AmountOutMinimum` from a fresh quote and an explicit short deadline.
3. Contract addresses and PoolKeys are immutable inputs to a trade decision; do not infer them from token symbols or untrusted APIs.
4. `minHopPriceX36` defaults to zero because Robinhood's router requires the field. `AmountOutMinimum` remains the primary aggregate slippage bound.
5. Some Long Doppler pools can restrict routers during an initial window. A revert is not evidence that the PoolKey should be changed.
6. The SDK returns unsigned call data. Nonce management, EIP-1559 fees, key custody, signing, simulation, retry policy, and submission remain with the caller.

## Development

```bash
go test ./...
go test -race ./...
go vet ./...
```

## License

MIT
