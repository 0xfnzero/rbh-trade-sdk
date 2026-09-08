package rbhtrade

import "testing"

func TestSupportedProtocolCapabilities(t *testing.T) {
	protocols := SupportedProtocols()
	if len(protocols) != 6 {
		t.Fatalf("supported protocol count = %d", len(protocols))
	}
	protocols[0].Capabilities.PredictLaunchAddresses = false
	if !SupportedProtocols()[0].Capabilities.PredictLaunchAddresses {
		t.Fatal("SupportedProtocols returned mutable package state")
	}
	pons, err := ParseProtocol(" PONS-V2 ")
	if err != nil || pons != ProtocolPonsV2 {
		t.Fatalf("ParseProtocol = %s, %v", pons, err)
	}
	if _, err := ParseProtocol("unknown"); err == nil {
		t.Fatal("unknown protocol was accepted")
	}
	if !SupportedProtocols()[0].Capabilities.QuoteCurveLocally || SupportedProtocols()[5].Capabilities.QuoteCurveLocally {
		t.Fatal("local quote capabilities do not reflect verified formulas")
	}
	if !SupportedProtocols()[2].Capabilities.QuoteHookedV4Locally || !SupportedProtocols()[5].Capabilities.QuoteHookedV4Locally {
		t.Fatal("verified o1 and Bags hook quote capabilities are missing")
	}
}
