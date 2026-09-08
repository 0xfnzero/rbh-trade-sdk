package pons

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const tokenParamsComponents = `[{"name":"name","type":"string"},{"name":"symbol","type":"string"},{"name":"logo","type":"string"},{"name":"description","type":"string"},{"name":"socials","type":"tuple","components":[{"name":"twitter","type":"string"},{"name":"telegram","type":"string"},{"name":"discord","type":"string"},{"name":"website","type":"string"},{"name":"farcaster","type":"string"}]},{"name":"creatorFeeRecipient","type":"address"},{"name":"creatorTaxBps","type":"uint16"},{"name":"buybackEnabled","type":"bool"},{"name":"expectedEconomics","type":"bytes32"},{"name":"salt","type":"bytes32"}]`

const launchConfigComponents = `[{"name":"supply","type":"uint256"},{"name":"curveFeeBps","type":"uint256"},{"name":"phantomQuote","type":"uint256"},{"name":"graduationThreshold","type":"uint256"},{"name":"poolFee","type":"uint24"},{"name":"tickSpacing","type":"int24"},{"name":"enabled","type":"bool"}]`

const feePolicyComponents = `[{"name":"protocolFeeRecipient","type":"address"},{"name":"protocolFeeShareBps","type":"uint16"},{"name":"buybackBurnBps","type":"uint16"},{"name":"hookFeeBps","type":"uint16"},{"name":"maxInternalPriceImpactBps","type":"uint16"}]`

const factoryABIJSON = `[
	{"type":"function","name":"CREATOR_FEE_RECIPIENT_TIMELOCK","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"CREATOR_FEE_RECIPIENT_EXECUTION_WINDOW","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"GRADUATION_RESCUE_DELAY","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"owner","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"pendingOwner","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"poolManager","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"positionManager","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"permit2","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"locker","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"memeHook","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"feeEscrow","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"buybackVault","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"graduationExecutor","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"launchDeployer","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"launchForwarder","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"graduationGuard","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"launchConfigCount","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"getLaunchConfig","stateMutability":"view","inputs":[{"name":"id","type":"uint256"}],"outputs":[{"name":"","type":"tuple","components":` + launchConfigComponents + `}]},
	{"type":"function","name":"launchFee","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"launchEnabled","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"bool"}]},
	{"type":"function","name":"maxCreatorTaxBps","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"snipeTaxStartBps","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"snipeTaxSeconds","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"canLaunch","stateMutability":"view","inputs":[{"name":"launcher","type":"address"}],"outputs":[{"name":"","type":"bool"}]},
	{"type":"function","name":"whitelistedLaunchers","stateMutability":"view","inputs":[{"name":"launcher","type":"address"}],"outputs":[{"name":"","type":"bool"}]},
	{"type":"function","name":"approvedPairTokens","stateMutability":"view","inputs":[{"name":"pairToken","type":"address"}],"outputs":[{"name":"","type":"bool"}]},
	{"type":"function","name":"pairTokenEconomics","stateMutability":"view","inputs":[{"name":"pairToken","type":"address"}],"outputs":[{"name":"phantomQuote","type":"uint256"},{"name":"graduationThreshold","type":"uint256"},{"name":"decimals","type":"uint8"}]},
	{"type":"function","name":"pendingCreatorFeeRecipient","stateMutability":"view","inputs":[{"name":"token","type":"address"}],"outputs":[{"name":"newRecipient","type":"address"},{"name":"effectiveAt","type":"uint256"},{"name":"expiresAt","type":"uint256"}]},
	{"type":"function","name":"getLaunchedToken","stateMutability":"view","inputs":[{"name":"token","type":"address"}],"outputs":[{"name":"","type":"tuple","components":[{"name":"token","type":"address"},{"name":"curve","type":"address"},{"name":"deployer","type":"address"},{"name":"creatorFeeRecipient","type":"address"},{"name":"pairToken","type":"address"},{"name":"graduationThreshold","type":"uint256"},{"name":"poolFee","type":"uint24"},{"name":"tickSpacing","type":"int24"},{"name":"creatorTaxBps","type":"uint16"},{"name":"buybackEnabled","type":"bool"},{"name":"phase","type":"uint8"},{"name":"sweptQuote","type":"uint256"},{"name":"sweptTokens","type":"uint256"},{"name":"sweptAt","type":"uint256"},{"name":"exists","type":"bool"}]}]},
	{"type":"function","name":"getLaunchFeePolicy","stateMutability":"view","inputs":[{"name":"token","type":"address"}],"outputs":[{"name":"","type":"tuple","components":` + feePolicyComponents + `}]},
	{"type":"function","name":"previewLaunchEconomics","stateMutability":"view","inputs":[{"name":"launchConfigId","type":"uint256"},{"name":"pairToken","type":"address"}],"outputs":[{"name":"","type":"bytes32"}]},

	{"type":"event","name":"TokenLaunched","inputs":[{"name":"token","type":"address","indexed":true},{"name":"curve","type":"address","indexed":true},{"name":"deployer","type":"address","indexed":true},{"name":"pairToken","type":"address","indexed":false},{"name":"launchConfigId","type":"uint256","indexed":false},{"name":"graduationThreshold","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"LaunchSwept","inputs":[{"name":"token","type":"address","indexed":true},{"name":"quoteOut","type":"uint256","indexed":false},{"name":"tokenOut","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"LaunchForceSwept","inputs":[{"name":"token","type":"address","indexed":true}],"anonymous":false},
	{"type":"event","name":"CreatorFeeRecipientUpdated","inputs":[{"name":"token","type":"address","indexed":true},{"name":"previousRecipient","type":"address","indexed":true},{"name":"newRecipient","type":"address","indexed":true}],"anonymous":false},
	{"type":"event","name":"CreatorFeeRecipientChangeProposed","inputs":[{"name":"token","type":"address","indexed":true},{"name":"currentRecipient","type":"address","indexed":true},{"name":"proposedRecipient","type":"address","indexed":true},{"name":"effectiveAt","type":"uint256","indexed":false},{"name":"expiresAt","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"CreatorFeeRecipientChangeCancelled","inputs":[{"name":"token","type":"address","indexed":true},{"name":"proposedRecipient","type":"address","indexed":true}],"anonymous":false},
	{"type":"event","name":"PoolGraduated","inputs":[{"name":"token","type":"address","indexed":true},{"name":"positionId","type":"uint256","indexed":false},{"name":"tokenAmount","type":"uint256","indexed":false},{"name":"pairTokenAmount","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"LaunchConfigAdded","inputs":[{"name":"id","type":"uint256","indexed":true}],"anonymous":false},
	{"type":"event","name":"LaunchConfigUpdated","inputs":[{"name":"id","type":"uint256","indexed":true}],"anonymous":false},
	{"type":"event","name":"LaunchFeeUpdated","inputs":[{"name":"launchFee","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"LaunchEnabledUpdated","inputs":[{"name":"enabled","type":"bool","indexed":false}],"anonymous":false},
	{"type":"event","name":"WhitelistedLauncherUpdated","inputs":[{"name":"launcher","type":"address","indexed":true},{"name":"enabled","type":"bool","indexed":false}],"anonymous":false},
	{"type":"event","name":"MaxCreatorTaxUpdated","inputs":[{"name":"bps","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"SnipeTaxStartBpsUpdated","inputs":[{"name":"bps","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"SnipeTaxSecondsUpdated","inputs":[{"name":"secondsWindow","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"GraduationExecutorSet","inputs":[{"name":"executor","type":"address","indexed":false}],"anonymous":false},
	{"type":"event","name":"LaunchDeployerSet","inputs":[{"name":"deployer","type":"address","indexed":false}],"anonymous":false},
	{"type":"event","name":"LaunchForwarderSet","inputs":[{"name":"forwarder","type":"address","indexed":false}],"anonymous":false},
	{"type":"event","name":"PairTokenApprovalUpdated","inputs":[{"name":"pairToken","type":"address","indexed":true},{"name":"approved","type":"bool","indexed":false}],"anonymous":false},
	{"type":"event","name":"PairTokenEconomicsUpdated","inputs":[{"name":"pairToken","type":"address","indexed":true},{"name":"phantomQuote","type":"uint256","indexed":false},{"name":"graduationThreshold","type":"uint256","indexed":false},{"name":"decimals","type":"uint8","indexed":false}],"anonymous":false},
	{"type":"event","name":"BuybackEnabledUpdated","inputs":[{"name":"token","type":"address","indexed":true},{"name":"enabled","type":"bool","indexed":false},{"name":"controller","type":"address","indexed":true}],"anonymous":false},
	{"type":"event","name":"GraduationTokensPermanentlyLocked","inputs":[{"name":"token","type":"address","indexed":true},{"name":"amount","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"LaunchGraduationRescued","inputs":[{"name":"token","type":"address","indexed":true},{"name":"recipient","type":"address","indexed":true},{"name":"quoteAmount","type":"uint256","indexed":false},{"name":"tokenAmount","type":"uint256","indexed":false}],"anonymous":false}
]`

const factoryWriteABIJSON = `[
	{"type":"function","name":"addLaunchConfig","stateMutability":"nonpayable","inputs":[{"name":"config","type":"tuple","components":` + launchConfigComponents + `}],"outputs":[{"name":"id","type":"uint256"}]},
	{"type":"function","name":"updateLaunchConfig","stateMutability":"nonpayable","inputs":[{"name":"id","type":"uint256"},{"name":"config","type":"tuple","components":` + launchConfigComponents + `}],"outputs":[]},
	{"type":"function","name":"setLaunchFee","stateMutability":"nonpayable","inputs":[{"name":"newLaunchFee","type":"uint256"}],"outputs":[]},
	{"type":"function","name":"setLaunchEnabled","stateMutability":"nonpayable","inputs":[{"name":"enabled","type":"bool"}],"outputs":[]},
	{"type":"function","name":"setWhitelistedLauncher","stateMutability":"nonpayable","inputs":[{"name":"launcher","type":"address"},{"name":"enabled","type":"bool"}],"outputs":[]},
	{"type":"function","name":"setPairTokenEconomics","stateMutability":"nonpayable","inputs":[{"name":"pairToken","type":"address"},{"name":"phantomQuote","type":"uint256"},{"name":"graduationThreshold","type":"uint256"},{"name":"expectedDecimals","type":"uint8"}],"outputs":[]},
	{"type":"function","name":"setPairTokenApproved","stateMutability":"nonpayable","inputs":[{"name":"pairToken","type":"address"},{"name":"approved","type":"bool"}],"outputs":[]},
	{"type":"function","name":"setMaxCreatorTaxBps","stateMutability":"nonpayable","inputs":[{"name":"bps","type":"uint256"}],"outputs":[]},
	{"type":"function","name":"setSnipeTaxStartBps","stateMutability":"nonpayable","inputs":[{"name":"bps","type":"uint256"}],"outputs":[]},
	{"type":"function","name":"setSnipeTaxSeconds","stateMutability":"nonpayable","inputs":[{"name":"secondsWindow","type":"uint256"}],"outputs":[]},
	{"type":"function","name":"setGraduationExecutor","stateMutability":"nonpayable","inputs":[{"name":"executor","type":"address"}],"outputs":[]},
	{"type":"function","name":"setLaunchDeployer","stateMutability":"nonpayable","inputs":[{"name":"deployer","type":"address"}],"outputs":[]},
	{"type":"function","name":"setLaunchForwarder","stateMutability":"nonpayable","inputs":[{"name":"forwarder","type":"address"}],"outputs":[]},
	{"type":"function","name":"launchTokenFor","stateMutability":"payable","inputs":[{"name":"params","type":"tuple","components":` + tokenParamsComponents + `},{"name":"launchConfigId","type":"uint256"},{"name":"pairToken","type":"address"},{"name":"originalDeployer","type":"address"},{"name":"snipeTaxExemptions","type":"address[]"}],"outputs":[{"name":"token","type":"address"},{"name":"curve","type":"address"}]},
	{"type":"function","name":"transferCreatorFeeRecipient","stateMutability":"nonpayable","inputs":[{"name":"token","type":"address"},{"name":"newRecipient","type":"address"}],"outputs":[]},
	{"type":"function","name":"setBuybackEnabled","stateMutability":"nonpayable","inputs":[{"name":"token","type":"address"},{"name":"enabled","type":"bool"}],"outputs":[]},
	{"type":"function","name":"setCreatorFeeRecipient","stateMutability":"nonpayable","inputs":[{"name":"token","type":"address"},{"name":"newRecipient","type":"address"}],"outputs":[]},
	{"type":"function","name":"executeCreatorFeeRecipientChange","stateMutability":"nonpayable","inputs":[{"name":"token","type":"address"}],"outputs":[]},
	{"type":"function","name":"cancelCreatorFeeRecipientChange","stateMutability":"nonpayable","inputs":[{"name":"token","type":"address"}],"outputs":[]},
	{"type":"function","name":"graduate","stateMutability":"nonpayable","inputs":[{"name":"token","type":"address"}],"outputs":[]},
	{"type":"function","name":"forceSweptGraduation","stateMutability":"nonpayable","inputs":[{"name":"token","type":"address"}],"outputs":[]},
	{"type":"function","name":"createGraduatedPool","stateMutability":"nonpayable","inputs":[{"name":"token","type":"address"}],"outputs":[{"name":"positionId","type":"uint256"}]},
	{"type":"function","name":"rescueCurveFees","stateMutability":"nonpayable","inputs":[{"name":"token","type":"address"}],"outputs":[]},
	{"type":"function","name":"rescueSweptGraduation","stateMutability":"nonpayable","inputs":[{"name":"token","type":"address"},{"name":"recipient","type":"address"}],"outputs":[]},
	{"type":"function","name":"transferOwnership","stateMutability":"nonpayable","inputs":[{"name":"newOwner","type":"address"}],"outputs":[]},
	{"type":"function","name":"acceptOwnership","stateMutability":"nonpayable","inputs":[],"outputs":[]}
]`

const factoryLaunchSimpleABIJSON = `[
	{"type":"function","name":"launchToken","stateMutability":"payable","inputs":[{"name":"params","type":"tuple","components":` + tokenParamsComponents + `},{"name":"launchConfigId","type":"uint256"},{"name":"pairToken","type":"address"}],"outputs":[{"name":"token","type":"address"},{"name":"curve","type":"address"}]}
]`

const factoryLaunchABIJSON = `[
	{"type":"function","name":"launchToken","stateMutability":"payable","inputs":[{"name":"params","type":"tuple","components":` + tokenParamsComponents + `},{"name":"launchConfigId","type":"uint256"},{"name":"pairToken","type":"address"},{"name":"snipeTaxExemptions","type":"address[]"}],"outputs":[{"name":"token","type":"address"},{"name":"curve","type":"address"}]}
]`

const routerABIJSON = `[
	{"type":"function","name":"launchAndBuy","stateMutability":"payable","inputs":[{"name":"params","type":"tuple","components":` + tokenParamsComponents + `},{"name":"launchConfigId","type":"uint256"},{"name":"pairToken","type":"address"},{"name":"quoteIn","type":"uint256"},{"name":"minTokensOut","type":"uint256"},{"name":"recipient","type":"address"},{"name":"snipeTaxExemptions","type":"address[]"}],"outputs":[{"name":"token","type":"address"},{"name":"curve","type":"address"},{"name":"tokensOut","type":"uint256"}]}
]`

const curveABIJSON = `[
	{"type":"function","name":"token","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"pairToken","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"deployer","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"factory","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"feePolicy","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"feeEscrow","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"buybackVault","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"protocolFeeRecipient","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"buybackCreatorRecipient","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"protocolFeeShareBps","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"buybackBurnBps","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"maxInternalPriceImpactBps","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"phantomQuote","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"feeBps","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"creatorTaxBps","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"graduationThreshold","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"buybackEnabled","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"bool"}]},
	{"type":"function","name":"quoteFeeBalance","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"buybackQuoteBalance","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"creatorTaxBalance","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"trackedQuote","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"trackedTokens","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"graduated","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"bool"}]},
	{"type":"function","name":"reservedTokens","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"isNativeQuote","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"bool"}]},
	{"type":"function","name":"getReserves","stateMutability":"view","inputs":[],"outputs":[{"name":"quoteReserve","type":"uint256"},{"name":"tokenReserve","type":"uint256"}]},
	{"type":"function","name":"quoteReserve","stateMutability":"view","inputs":[],"outputs":[{"name":"quoteReserve","type":"uint256"}]},
	{"type":"function","name":"realQuoteReserve","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"tokenReserve","stateMutability":"view","inputs":[],"outputs":[{"name":"tokenReserve","type":"uint256"}]},
	{"type":"function","name":"sellableTokens","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"readyToGraduate","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"bool"}]},
	{"type":"function","name":"currentSnipeTaxBps","stateMutability":"view","inputs":[{"name":"recipient","type":"address"}],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"exemptFromSnipeTax","stateMutability":"nonpayable","inputs":[{"name":"recipient","type":"address"}],"outputs":[]},
	{"type":"function","name":"buy","stateMutability":"payable","inputs":[{"name":"quoteIn","type":"uint256"},{"name":"minTokensOut","type":"uint256"},{"name":"recipient","type":"address"}],"outputs":[{"name":"tokensOut","type":"uint256"}]},
	{"type":"function","name":"sell","stateMutability":"nonpayable","inputs":[{"name":"tokensIn","type":"uint256"},{"name":"minQuoteOut","type":"uint256"},{"name":"recipient","type":"address"}],"outputs":[{"name":"quoteOut","type":"uint256"}]},
	{"type":"function","name":"sweepFees","stateMutability":"nonpayable","inputs":[{"name":"minBuybackTokensOut","type":"uint256"}],"outputs":[]},
	{"type":"function","name":"graduate","stateMutability":"nonpayable","inputs":[{"name":"recipient","type":"address"}],"outputs":[{"name":"quoteOut","type":"uint256"},{"name":"tokenOut","type":"uint256"}]},
	{"type":"function","name":"rescueFees","stateMutability":"nonpayable","inputs":[],"outputs":[{"name":"protocolAmount","type":"uint256"},{"name":"creatorAmount","type":"uint256"}]},

	{"type":"event","name":"CurveBuy","inputs":[{"name":"buyer","type":"address","indexed":true},{"name":"recipient","type":"address","indexed":true},{"name":"quoteIn","type":"uint256","indexed":false},{"name":"tokensOut","type":"uint256","indexed":false},{"name":"fee","type":"uint256","indexed":false},{"name":"tax","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"CurveBuyRefunded","inputs":[{"name":"buyer","type":"address","indexed":true},{"name":"refund","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"CurveSell","inputs":[{"name":"seller","type":"address","indexed":true},{"name":"recipient","type":"address","indexed":true},{"name":"tokensIn","type":"uint256","indexed":false},{"name":"quoteOut","type":"uint256","indexed":false},{"name":"fee","type":"uint256","indexed":false},{"name":"tax","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"FeesSwept","inputs":[{"name":"protocolAmount","type":"uint256","indexed":false},{"name":"buybackAmount","type":"uint256","indexed":false},{"name":"creatorAmount","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"FeesRescued","inputs":[{"name":"protocolRecipient","type":"address","indexed":true},{"name":"creatorRecipient","type":"address","indexed":true},{"name":"protocolAmount","type":"uint256","indexed":false},{"name":"creatorAmount","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"BuybackLocked","inputs":[{"name":"quoteSpent","type":"uint256","indexed":false},{"name":"tokensLocked","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"CurveCompleted","inputs":[{"name":"recipient","type":"address","indexed":false},{"name":"quoteOut","type":"uint256","indexed":false},{"name":"tokenOut","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"Initialized","inputs":[{"name":"token","type":"address","indexed":false}],"anonymous":false},
	{"type":"event","name":"CreatorFeeRecipientUpdated","inputs":[{"name":"previousRecipient","type":"address","indexed":true},{"name":"newRecipient","type":"address","indexed":true}],"anonymous":false},
	{"type":"event","name":"BuybackEnabledUpdated","inputs":[{"name":"enabled","type":"bool","indexed":false}],"anonymous":false},
	{"type":"event","name":"AutoGraduationFailed","inputs":[{"name":"token","type":"address","indexed":true},{"name":"gasRemaining","type":"uint256","indexed":false}],"anonymous":false}
]`

const escrowABIJSON = `[
	{"type":"function","name":"credit","stateMutability":"payable","inputs":[{"name":"recipient","type":"address"}],"outputs":[]},
	{"type":"function","name":"creditToken","stateMutability":"nonpayable","inputs":[{"name":"recipient","type":"address"},{"name":"token","type":"address"},{"name":"amount","type":"uint256"}],"outputs":[]},
	{"type":"function","name":"claim","stateMutability":"nonpayable","inputs":[],"outputs":[{"name":"amount","type":"uint256"}]},
	{"type":"function","name":"claimToken","stateMutability":"nonpayable","inputs":[{"name":"token","type":"address"}],"outputs":[{"name":"amount","type":"uint256"}]},
	{"type":"function","name":"balanceOf","stateMutability":"view","inputs":[{"name":"recipient","type":"address"}],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"balanceOfToken","stateMutability":"view","inputs":[{"name":"recipient","type":"address"},{"name":"token","type":"address"}],"outputs":[{"name":"","type":"uint256"}]}
]`

const escrowClaimAmountABIJSON = `[
	{"type":"function","name":"claim","stateMutability":"nonpayable","inputs":[{"name":"amount","type":"uint256"}],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"claimToken","stateMutability":"nonpayable","inputs":[{"name":"token","type":"address"},{"name":"amount","type":"uint256"}],"outputs":[{"name":"","type":"uint256"}]}
]`

const tokenABIJSON = `[
	{"type":"function","name":"deployer","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"launchFactory","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"curve","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"logo","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"string"}]},
	{"type":"function","name":"description","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"string"}]},
	{"type":"function","name":"socials","stateMutability":"view","inputs":[],"outputs":[{"name":"twitter","type":"string"},{"name":"telegram","type":"string"},{"name":"discord","type":"string"},{"name":"website","type":"string"},{"name":"farcaster","type":"string"}]},
	{"type":"function","name":"getTokenInfo","stateMutability":"view","inputs":[],"outputs":[{"name":"tokenDeployer","type":"address"},{"name":"tokenLogo","type":"string"},{"name":"tokenDescription","type":"string"},{"name":"tokenSocials","type":"tuple","components":[{"name":"twitter","type":"string"},{"name":"telegram","type":"string"},{"name":"discord","type":"string"},{"name":"website","type":"string"},{"name":"farcaster","type":"string"}]}]},
	{"type":"function","name":"name","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"string"}]},
	{"type":"function","name":"symbol","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"string"}]},
	{"type":"function","name":"decimals","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint8"}]},
	{"type":"function","name":"totalSupply","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"balanceOf","stateMutability":"view","inputs":[{"name":"account","type":"address"}],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"allowance","stateMutability":"view","inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"approve","stateMutability":"nonpayable","inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"outputs":[{"name":"","type":"bool"}]},
	{"type":"function","name":"transfer","stateMutability":"nonpayable","inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"outputs":[{"name":"","type":"bool"}]},
	{"type":"function","name":"transferFrom","stateMutability":"nonpayable","inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"outputs":[{"name":"","type":"bool"}]},
	{"type":"function","name":"burn","stateMutability":"nonpayable","inputs":[{"name":"value","type":"uint256"}],"outputs":[]},
	{"type":"function","name":"burnFrom","stateMutability":"nonpayable","inputs":[{"name":"account","type":"address"},{"name":"value","type":"uint256"}],"outputs":[]},
	{"type":"event","name":"Transfer","inputs":[{"name":"from","type":"address","indexed":true},{"name":"to","type":"address","indexed":true},{"name":"value","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"Approval","inputs":[{"name":"owner","type":"address","indexed":true},{"name":"spender","type":"address","indexed":true},{"name":"value","type":"uint256","indexed":false}],"anonymous":false}
]`

const vaultABIJSON = `[
	{"type":"function","name":"feePolicy","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"feeEscrow","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"factory","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"VESTING_DURATION","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"totalLocked","stateMutability":"view","inputs":[{"name":"token","type":"address"}],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"totalReleased","stateMutability":"view","inputs":[{"name":"token","type":"address"}],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"vestingStart","stateMutability":"view","inputs":[{"name":"token","type":"address"}],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"vestedAmount","stateMutability":"view","inputs":[{"name":"token","type":"address"}],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"releasable","stateMutability":"view","inputs":[{"name":"token","type":"address"}],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"vestingTerms","stateMutability":"view","inputs":[{"name":"token","type":"address"}],"outputs":[{"name":"creatorRecipient","type":"address"},{"name":"protocolRecipient","type":"address"},{"name":"protocolFeeShareBps","type":"uint16"}]},
	{"type":"function","name":"release","stateMutability":"nonpayable","inputs":[{"name":"token","type":"address"}],"outputs":[{"name":"released","type":"uint256"}]},
	{"type":"event","name":"FactorySet","inputs":[{"name":"factory","type":"address","indexed":false}],"anonymous":false},
	{"type":"event","name":"Locked","inputs":[{"name":"token","type":"address","indexed":true},{"name":"depositor","type":"address","indexed":true},{"name":"amount","type":"uint256","indexed":false},{"name":"newVestingStart","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"VestingTermsSnapshotted","inputs":[{"name":"token","type":"address","indexed":true},{"name":"creatorRecipient","type":"address","indexed":true},{"name":"protocolRecipient","type":"address","indexed":true},{"name":"protocolFeeShareBps","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"Released","inputs":[{"name":"token","type":"address","indexed":true},{"name":"creatorAmount","type":"uint256","indexed":false},{"name":"protocolAmount","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"CreatorRecipientUpdated","inputs":[{"name":"token","type":"address","indexed":true},{"name":"previousRecipient","type":"address","indexed":true},{"name":"newRecipient","type":"address","indexed":true}],"anonymous":false}
]`

const hookABIJSON = `[
	{"type":"function","name":"feeEscrow","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"factory","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"buybackVault","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"protocolFeeRecipient","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"protocolFeeShareBps","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"buybackBurnBps","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"hookFeeBps","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"maxInternalPriceImpactBps","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"feeSweepOperator","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"currentFeePolicy","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"tuple","components":` + feePolicyComponents + `}]},
	{"type":"function","name":"launches","stateMutability":"view","inputs":[{"name":"poolId","type":"bytes32"}],"outputs":[{"name":"registered","type":"bool"},{"name":"memecoinIsCurrency0","type":"bool"},{"name":"memecoin","type":"address"},{"name":"quoteToken","type":"address"},{"name":"creator","type":"address"},{"name":"buybackCreatorRecipient","type":"address"},{"name":"protocolFeeRecipient","type":"address"},{"name":"creatorTaxBps","type":"uint16"},{"name":"protocolFeeShareBps","type":"uint16"},{"name":"buybackBurnBps","type":"uint16"},{"name":"hookFeeBps","type":"uint16"},{"name":"maxInternalPriceImpactBps","type":"uint16"},{"name":"buybackEnabled","type":"bool"}]},
	{"type":"function","name":"pendingFees","stateMutability":"view","inputs":[{"name":"poolId","type":"bytes32"},{"name":"currency","type":"address"}],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"pendingCreatorTax","stateMutability":"view","inputs":[{"name":"poolId","type":"bytes32"},{"name":"currency","type":"address"}],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"pendingBuyback","stateMutability":"view","inputs":[{"name":"poolId","type":"bytes32"},{"name":"currency","type":"address"}],"outputs":[{"name":"","type":"uint256"}]},
	{"type":"function","name":"sweepPoolFees","stateMutability":"nonpayable","inputs":[{"name":"poolId","type":"bytes32"},{"name":"minConversionQuoteOut","type":"uint256"},{"name":"minBuybackTokensOut","type":"uint256"}],"outputs":[]},
	{"type":"function","name":"rescuePoolFees","stateMutability":"nonpayable","inputs":[{"name":"poolId","type":"bytes32"}],"outputs":[{"name":"protocolAmount","type":"uint256"},{"name":"creatorAmount","type":"uint256"}]},

	{"type":"event","name":"FactorySet","inputs":[{"name":"factory","type":"address","indexed":false}],"anonymous":false},
	{"type":"event","name":"PoolRegistered","inputs":[{"name":"poolId","type":"bytes32","indexed":true},{"name":"memecoin","type":"address","indexed":false},{"name":"quoteToken","type":"address","indexed":false},{"name":"creator","type":"address","indexed":false}],"anonymous":false},
	{"type":"event","name":"CreatorFeeRecipientUpdated","inputs":[{"name":"poolId","type":"bytes32","indexed":true},{"name":"previousRecipient","type":"address","indexed":true},{"name":"newRecipient","type":"address","indexed":true}],"anonymous":false},
	{"type":"event","name":"HookFeeCollected","inputs":[{"name":"poolId","type":"bytes32","indexed":true},{"name":"currency","type":"address","indexed":false},{"name":"feeAmount","type":"uint256","indexed":false},{"name":"taxAmount","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"PoolFeesSwept","inputs":[{"name":"poolId","type":"bytes32","indexed":true},{"name":"protocolAmount","type":"uint256","indexed":false},{"name":"buybackAmount","type":"uint256","indexed":false},{"name":"creatorAmount","type":"uint256","indexed":false},{"name":"tokensLocked","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"PoolFeesRescued","inputs":[{"name":"poolId","type":"bytes32","indexed":true},{"name":"quoteToken","type":"address","indexed":true},{"name":"protocolAmount","type":"uint256","indexed":false},{"name":"creatorAmount","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"PoolBuybackSkipped","inputs":[{"name":"poolId","type":"bytes32","indexed":true},{"name":"foldedBackQuote","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"PoolConversionSkipped","inputs":[{"name":"poolId","type":"bytes32","indexed":true},{"name":"retainedMemecoin","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"BuybackVaultSet","inputs":[{"name":"vault","type":"address","indexed":false}],"anonymous":false},
	{"type":"event","name":"ProtocolFeeShareUpdated","inputs":[{"name":"bps","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"BuybackBurnBpsUpdated","inputs":[{"name":"bps","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"HookFeeBpsUpdated","inputs":[{"name":"bps","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"MaxInternalPriceImpactUpdated","inputs":[{"name":"bps","type":"uint256","indexed":false}],"anonymous":false},
	{"type":"event","name":"ProtocolFeeRecipientUpdated","inputs":[{"name":"recipient","type":"address","indexed":false}],"anonymous":false},
	{"type":"event","name":"FeeSweepOperatorUpdated","inputs":[{"name":"operator","type":"address","indexed":false}],"anonymous":false},
	{"type":"event","name":"BuybackEnabledUpdated","inputs":[{"name":"poolId","type":"bytes32","indexed":true},{"name":"enabled","type":"bool","indexed":false}],"anonymous":false}
]`

const lockerABIJSON = `[
	{"type":"function","name":"factory","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"positionManager","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"type":"function","name":"isLocked","stateMutability":"view","inputs":[{"name":"token","type":"address"}],"outputs":[{"name":"","type":"bool"}]},
	{"type":"event","name":"FactorySet","inputs":[{"name":"factory","type":"address","indexed":false}],"anonymous":false},
	{"type":"event","name":"PositionLocked","inputs":[{"name":"token","type":"address","indexed":true},{"name":"tokenId","type":"uint256","indexed":true}],"anonymous":false},
	{"type":"event","name":"TokenSupplyLocked","inputs":[{"name":"token","type":"address","indexed":true},{"name":"amount","type":"uint256","indexed":false}],"anonymous":false}
]`

var (
	factoryContractABI, FactoryABI                   = mustABIPair(factoryABIJSON)
	factoryWriteContractABI, FactoryWriteABI         = mustABIPair(factoryWriteABIJSON)
	factorySimpleContractABI, FactoryLaunchSimpleABI = mustABIPair(factoryLaunchSimpleABIJSON)
	factoryLaunchContractABI, FactoryLaunchABI       = mustABIPair(factoryLaunchABIJSON)
	routerContractABI, RouterABI                     = mustABIPair(routerABIJSON)
	curveContractABI, CurveABI                       = mustABIPair(curveABIJSON)
	escrowContractABI, EscrowABI                     = mustABIPair(escrowABIJSON)
	escrowAmountContractABI, EscrowClaimAmountABI    = mustABIPair(escrowClaimAmountABIJSON)
	tokenContractABI, TokenABI                       = mustABIPair(tokenABIJSON)
	vaultContractABI, VaultABI                       = mustABIPair(vaultABIJSON)
	hookContractABI, HookABI                         = mustABIPair(hookABIJSON)
	lockerContractABI, LockerABI                     = mustABIPair(lockerABIJSON)
)

func mustABIPair(raw string) (abi.ABI, abi.ABI) {
	return mustABI(raw), mustABI(raw)
}

func mustABI(raw string) abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(raw))
	if err != nil {
		panic(err)
	}
	return parsed
}
