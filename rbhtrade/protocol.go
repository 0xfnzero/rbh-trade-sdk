package rbhtrade

import (
	"fmt"
	"strings"
)

type Protocol uint8

const (
	ProtocolUnknown Protocol = iota
	ProtocolPonsV2
	ProtocolLong
	ProtocolO1
	ProtocolPoolsTrade
	ProtocolPAIR
	ProtocolBagsV2
)

func (p Protocol) String() string {
	switch p {
	case ProtocolPonsV2:
		return "pons-v2"
	case ProtocolLong:
		return "long"
	case ProtocolO1:
		return "o1"
	case ProtocolPoolsTrade:
		return "pools.trade"
	case ProtocolPAIR:
		return "pair"
	case ProtocolBagsV2:
		return "bags-v2"
	default:
		return "unknown"
	}
}

type Lifecycle uint8

const (
	LifecycleUnknown Lifecycle = iota
	LifecycleBondingCurveThenV4
	LifecycleDirectV4
	LifecycleDirectOrCrowdsaleThenV4
)

// Deprecated: use LifecycleDirectOrCrowdsaleThenV4 for Pools.trade.
const LifecycleCrowdsaleThenV4 = LifecycleDirectOrCrowdsaleThenV4

type ProtocolInfo struct {
	Protocol     Protocol
	Lifecycle    Lifecycle
	Capabilities ProtocolCapabilities
}

// ProtocolCapabilities separates generic calldata support from protocol
// formulas that have been verified strongly enough for local hot-path use.
// False values are production fail-closed boundaries, not roadmap promises.
type ProtocolCapabilities struct {
	BuildLaunchCall        bool
	BuildCurveSwap         bool
	BuildV4Swap            bool
	PredictLaunchAddresses bool
	QuoteCurveLocally      bool
	QuoteHookedV4Locally   bool
}

var supportedProtocols = [...]ProtocolInfo{
	{Protocol: ProtocolPonsV2, Lifecycle: LifecycleBondingCurveThenV4, Capabilities: ProtocolCapabilities{BuildLaunchCall: true, BuildCurveSwap: true, BuildV4Swap: true, PredictLaunchAddresses: true, QuoteCurveLocally: true, QuoteHookedV4Locally: true}},
	{Protocol: ProtocolLong, Lifecycle: LifecycleDirectV4, Capabilities: ProtocolCapabilities{BuildLaunchCall: true, BuildV4Swap: true}},
	{Protocol: ProtocolO1, Lifecycle: LifecycleDirectV4, Capabilities: ProtocolCapabilities{BuildLaunchCall: true, BuildV4Swap: true, QuoteHookedV4Locally: true}},
	{Protocol: ProtocolPoolsTrade, Lifecycle: LifecycleDirectOrCrowdsaleThenV4, Capabilities: ProtocolCapabilities{BuildLaunchCall: true, BuildV4Swap: true}},
	{Protocol: ProtocolPAIR, Lifecycle: LifecycleDirectV4, Capabilities: ProtocolCapabilities{BuildLaunchCall: true, BuildV4Swap: true}},
	{Protocol: ProtocolBagsV2, Lifecycle: LifecycleBondingCurveThenV4, Capabilities: ProtocolCapabilities{BuildLaunchCall: true, BuildCurveSwap: true, BuildV4Swap: true, QuoteHookedV4Locally: true}},
}

func SupportedProtocols() []ProtocolInfo {
	out := make([]ProtocolInfo, len(supportedProtocols))
	copy(out, supportedProtocols[:])
	return out
}

func ParseProtocol(s string) (Protocol, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	for _, info := range supportedProtocols {
		if info.Protocol.String() == s {
			return info.Protocol, nil
		}
	}
	return ProtocolUnknown, fmt.Errorf("unsupported protocol %q", s)
}
