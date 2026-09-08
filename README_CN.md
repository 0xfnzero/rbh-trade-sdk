<div align="center">
  <h1>RBH Trade SDK for Go</h1>
  <h3><em>面向 Robinhood Chain Launchpad 与 Uniswap v4 交易的低延迟 Go SDK</em></h3>
</div>

<p align="center">
  <strong>为 Pons V2、Long、o1、Pools.trade、PAIR、Bags V2、Permit2 和 Robinhood Chain Universal Router 提供类型安全的 calldata builder。</strong>
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

## SDK

| SDK | Go Module |
|-----|-----------|
| Trade | [`github.com/0xfnzero/rbh-trade-sdk`](https://github.com/0xfnzero/rbh-trade-sdk) |
| Parser | [`github.com/0xfnzero/rbh-parser-sdk`](https://github.com/0xfnzero/rbh-parser-sdk) |

## 平台覆盖

| 平台 | Launch 生命周期 | 交易路径 |
|------|-----------------|----------|
| Pons V2 | Bonding Curve，毕业后进入 Uniswap v4 | 毕业前使用内置 Pons client 与 Curve 报价，毕业后使用共享 v4 builder |
| Long | 通过 Doppler/Airlock 直接创建 Uniswap v4 池 | 使用 `Initialize` 中的完整 PoolKey |
| o1 | 创建时提供永久 Uniswap v4 流动性 | 共享 v4 builder，支持 hook data |
| Pools.trade | Instant v4，或 Crowd Sale 后进入 v4 | Pools launch builder 与共享 v4 builder |
| PAIR | 同时创建 1 至 5 个永久 v4 池 | PAIR 多市场 launch 与共享 v4 builder |
| Bags V2 | Bonding Curve，毕业后进入 Uniswap v4 | Bags Curve builder 与共享 v4 builder |

`SupportedProtocols()` 通过 `ProtocolCapabilities` 暴露 launch、curve、v4、
确定性地址预测和已验证本地定价能力。应用应按这些标志 fail closed，不能把“可以构造
calldata”误当成“已经能在执行前可靠报价”。

## 功能

- 内置 Robinhood Chain `4663` 部署地址，并支持启动时校验合约 bytecode。
- 正确编码 Robinhood 修改版 Universal Router v4 exact-input，包括必需的 `minHopPriceX36`。
- ERC-20 与 Permit2 授权 builder。
- V4 Quoter 和 StateView 的请求及返回值解码。
- 基于事件 tick/liquidity 状态的本地 V4 exact-input 报价，使用经审计的 Uniswap 核心数学，并支持输出侧 hook 费率扣减。
- 按 exact-match 验证源码实现 o1 LaunchHook 的 quote 资产费用和线性 anti-snipe 衰减公式：`BuildO1HookPoolConfig`、`DecodeO1HookPoolConfig`、`O1HookFeeBPS`、`QuoteO1V4ExactInput`；调用方必须预热池子的固定费率计划，并提供目标区块时间戳。
- 按 exact-match 验证源码实现 Bags V4 hook 报价：`QuoteBagsV4ExactInput` 将核心 LP 费覆盖为 0，并在 WETH 腿收取 2%。
- Long、o1、Pools.trade、PAIR、Bags V2 launch builder。
- 完整内置 Pons V2 ABI、类型化读取客户端、Curve 报价数学、calldata builder 与交易封装。
- 基于 Sourcify exact-match deployer bytecode 的 Pons CREATE2 确定性 launch 预测，并校验 economics commitment 与 runtime code hash。
- calldata builder 不执行隐式 RPC，也不接触私钥。

## 安装

### 克隆源码到项目中

```bash
cd your_project_root_directory
git clone --branch v0.1.0 --depth 1 https://github.com/0xfnzero/rbh-trade-sdk
```

在业务项目的 `go.mod` 中添加：

```go
require github.com/0xfnzero/rbh-trade-sdk v0.1.0

replace github.com/0xfnzero/rbh-trade-sdk => ./rbh-trade-sdk
```

然后执行：

```bash
go mod tidy
```

### 使用 Go Modules

```bash
go get github.com/0xfnzero/rbh-trade-sdk@v0.1.0
```

## 构建 v4 Swap

完整 `PoolKey` 必须来自 PoolManager `Initialize` 事件。对于 Long，绝不能把 `LaunchCreated.poolOrHook` 当成真正的 hook。

```go
token := common.HexToAddress("0xYourToken")
quote := common.HexToAddress("0xYourQuoteToken")

call, err := rbhtrade.BuildV4ExactInputSingle(rbhtrade.ExactInputRequest{
    PoolKey: rbhtrade.PoolKey{
        Currency0:   quote, // currency 必须按地址排序
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
```

`call.To`、`call.Data`、`call.Value` 是待签名交易参数，签名与发送由业务自己的交易栈负责。

ERC-20 输入需要先授权 Permit2，再由 Permit2 授权 Universal Router：

```go
approvals, err := rbhtrade.BuildUniversalRouterApprovals(
    quote,
    big.NewInt(1_000_000),
    uint64(time.Now().Add(24*time.Hour).Unix()),
)
```

原生币输入使用零地址作为 `CurrencyIn`。SDK 会自动令 `call.Value = AmountIn`，不需要 Permit2。

## 报价与池状态

```go
quoteCall, err := rbhtrade.BuildV4QuoteExactInputSingle(rbhtrade.QuoteExactInputRequest{
    PoolKey: key, CurrencyIn: quoteToken, AmountIn: amountIn, HookData: hookData,
})
// 对 quoteCall 执行 eth_call 后：
result, err := rbhtrade.DecodeV4QuoteExactInputSingle(returnData)

poolID, err := rbhtrade.PoolID(key)
slotCall, err := rbhtrade.BuildStateViewGetSlot0(poolID)
slot0, err := rbhtrade.DecodeStateViewSlot0(slotReturnData)
```

## Long Launch

`BuildLongCreate` 实现了已验证的 `LongLauncher.create`，selector 为 `0x882db707`。Doppler/Airlock 各 factory 的 payload 由调用方提供；SDK 会拒绝不是 Long 官方可信 TokenFactory 的参数。

```go
call, err := rbhtrade.BuildLongCreate(rbhtrade.LongCreateParams{
    InitialSupply:       initialSupply,
    NumTokensToSell:     tokensToSell,
    Numeraire:           stockOrQuoteToken,
    TokenFactory:        rbhtrade.DefaultAddressBook().LongTokenFactory,
    TokenFactoryData:    tokenFactoryData,
    GovernanceFactory:   governanceFactory,
    PoolInitializer:     poolInitializer,
    PoolInitializerData: poolInitializerData,
    LiquidityMigrator:   liquidityMigrator,
    Salt:                salt,
})
```

请使用配套 parser 将 `LaunchCreated` 与同一交易的 PoolManager `Initialize` 关联；真正的 PoolKey 只存在于后者。

## 其他 Launchpad

| 平台 | Builder | value 与资金规则 |
|------|---------|------------------|
| o1 | `BuildO1Launch`、`BuildO1LaunchAndBuy` | `value` 传入当前 launch fee；使用原生币买入时必须满足 `value >= AmountIn` |
| Pools.trade | `BuildPoolsCreateToken`、`BuildPoolsDistributeToken`、`BuildPoolsDistributeWithNative`、`BuildPoolsMulticall` | 使用协议实时返回的费用/资金值；空 multicall 会被拒绝 |
| PAIR | `BuildPAIRLaunch` | allocation 权重总和必须为 10,000 BPS；不执行 developer buy 时使用 `PAIRNoDeveloperBuy` 且买入金额为零 |
| Bags V2 | `BuildBagsCreate`、`BuildBagsCreateAndBuy`、`BuildBagsBuy`、`BuildBagsSell` | Curve 买入通过 `value` 支付；Curve 卖出所需 token approval 由 SDK 外部处理 |

构造调用前，应从目标协议实时读取 fee、config、quote 和 deadline 输入。Builder 会校验 ABI 数值范围与结构约束，但不会隐式读取可变链上配置。

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

Pons 包完全实现在当前 module 内，不再需要单独依赖 Pons SDK。

## 启动校验

```go
backend, err := ethclient.DialContext(ctx, rpcURL)
client, err := rbhtrade.NewClientChecked(ctx, backend)
```

`NewClientChecked` 会校验 chain id `4663`，并并发检查所有被 builder 直接使用的固定合约地址是否存在 bytecode。该操作应在启动阶段执行，不会进入交易热路径。

```bash
ROBINHOOD_RPC_URL=https://rpc.mainnet.chain.robinhood.com go test ./rbhtrade -run TestRobinhoodDeployment
```

## 安全说明

1. 所有带资金的调用在签名前都应基于目标区块执行 simulation 或 gas estimation。
2. 使用最新报价生成非零 `AmountOutMinimum`，并设置明确且较短的 deadline。
3. 合约地址与 PoolKey 是交易决策的不可变输入，不能根据 token symbol 或不可信 API 猜测。
4. `minHopPriceX36` 默认是零，因为 Robinhood Router 要求存在该字段；整体滑点保护仍由 `AmountOutMinimum` 承担。
5. 部分 Long Doppler 池在初始阶段可能限制 Router。交易 revert 不代表应该篡改 PoolKey。
6. SDK 只返回未签名调用数据。nonce、EIP-1559 fee、私钥、签名、simulation、重试与发送策略由调用方负责。

## 开发

```bash
go test ./...
go test -race ./...
go vet ./...
```

## License

MIT
