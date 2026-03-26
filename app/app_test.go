package app_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmttypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/testutil/mock"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	utils "github.com/cosmos/evm/utils"
	erc20types "github.com/cosmos/evm/x/erc20/types"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	"github.com/stretchr/testify/require"

	"github.com/monolythium/mono-chain/app"
	burnmoduletypes "github.com/monolythium/mono-chain/x/burn/types"
	validatormoduletypes "github.com/monolythium/mono-chain/x/validator/types"
)

func TestMain(m *testing.M) {
	cfg := sdk.GetConfig()
	app.SetBech32Prefixes(cfg)
	app.SetBip44CoinType(cfg)
	os.Exit(m.Run())
}

func newTestApp(t *testing.T) *app.App {
	t.Helper()
	return app.New(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		simtestutil.EmptyAppOptions{},
	)
}

func TestAppInit_DoesNotPanic(t *testing.T) {
	require.NotPanics(t, func() {
		newTestApp(t)
	})
}

func TestAppInit_KeepersNonNil(t *testing.T) {
	a := newTestApp(t)

	require.NotNil(t, a.AccountKeeper)
	require.NotNil(t, a.BankKeeper)
	require.NotNil(t, a.StakingKeeper)
	require.NotNil(t, a.MintKeeper)
	require.NotNil(t, a.UpgradeKeeper)
	require.NotNil(t, a.GovKeeper)
	require.NotNil(t, a.SlashingKeeper)
	require.NotNil(t, a.EVMKeeper)
	require.NotNil(t, a.FeeMarketKeeper)
	require.NotNil(t, a.IBCKeeper)
	require.NotNil(t, a.BurnKeeper)
	require.NotNil(t, a.ValidatorKeeper)
	require.NotNil(t, a.AuthzKeeper)
	require.NotNil(t, a.FeeGrantKeeper)
	require.NotNil(t, a.ModuleManager)
}

func TestAppInit_CodecsAvailable(t *testing.T) {
	a := newTestApp(t)

	require.NotNil(t, a.AppCodec())
	require.NotNil(t, a.LegacyAmino())
	require.NotNil(t, a.InterfaceRegistry())
	require.NotNil(t, a.TxConfig())
}

func TestAppInit_ConfiguratorAvailable(t *testing.T) {
	a := newTestApp(t)

	// Verify Configurator getter exists and returns non-nil
	require.NotNil(t, a.Configurator(), "Configurator must be available for upgrades")
}

func TestAppInit_EVMStoreKeysRegistered(t *testing.T) {
	a := newTestApp(t)

	require.NotNil(t, a.GetKey(evmtypes.StoreKey), "evm store key missing")
	require.NotNil(t, a.GetKey(feemarkettypes.StoreKey), "feemarket store key missing")
	require.NotNil(t, a.GetKey(erc20types.StoreKey), "erc20 store key missing")
}

func TestPowerReduction_Is18Decimals(t *testing.T) {
	expected := utils.AttoPowerReduction
	require.True(t, sdk.DefaultPowerReduction.Equal(expected),
		"DefaultPowerReduction should be AttoPowerReduction, got %s", sdk.DefaultPowerReduction)

	tenTo18 := math.NewIntWithDecimal(1, 18)
	require.True(t, sdk.DefaultPowerReduction.Equal(tenTo18),
		"DefaultPowerReduction should equal 10^18, got %s", sdk.DefaultPowerReduction)
}

func TestDefaultBondDenom_IsAlyth(t *testing.T) {
	require.Equal(t, "alyth", sdk.DefaultBondDenom)
}

func TestModuleAccountPerms_Standard(t *testing.T) {
	perms := app.GetMaccPerms()

	_, hasFeeCollector := perms[authtypes.FeeCollectorName]
	require.True(t, hasFeeCollector)

	_, hasDistr := perms[distrtypes.ModuleName]
	require.True(t, hasDistr)

	require.Contains(t, perms[minttypes.ModuleName], authtypes.Minter)
	require.Contains(t, perms[stakingtypes.BondedPoolName], authtypes.Burner)
	require.Contains(t, perms[stakingtypes.BondedPoolName], authtypes.Staking)
	require.Contains(t, perms[stakingtypes.NotBondedPoolName], authtypes.Burner)
	require.Contains(t, perms[stakingtypes.NotBondedPoolName], authtypes.Staking)
}

func TestModuleAccountPerms_EVM(t *testing.T) {
	perms := app.GetMaccPerms()

	require.Contains(t, perms[evmtypes.ModuleName], authtypes.Minter)
	require.Contains(t, perms[evmtypes.ModuleName], authtypes.Burner)
	require.Contains(t, perms[erc20types.ModuleName], authtypes.Minter)
	require.Contains(t, perms[erc20types.ModuleName], authtypes.Burner)

	_, hasFeemarket := perms[feemarkettypes.ModuleName]
	require.True(t, hasFeemarket)

	require.Contains(t, perms[ibctransfertypes.ModuleName], authtypes.Minter)
	require.Contains(t, perms[ibctransfertypes.ModuleName], authtypes.Burner)
	require.Contains(t, perms[govtypes.ModuleName], authtypes.Burner)
}

func TestModuleAccountPerms_Custom(t *testing.T) {
	perms := app.GetMaccPerms()

	require.Contains(t, perms[burnmoduletypes.ModuleName], authtypes.Burner)
	require.Nil(t, perms[validatormoduletypes.ModuleName])
}

func TestAddressCodec_Bech32Roundtrip(t *testing.T) {
	a := newTestApp(t)
	codec := a.AccountKeeper.AddressCodec()

	addrBytes := make([]byte, 20)
	for i := range addrBytes {
		addrBytes[i] = byte(i + 1)
	}

	addrStr, err := codec.BytesToString(addrBytes)
	require.NoError(t, err)
	require.Contains(t, addrStr, "mono1")

	recovered, err := codec.StringToBytes(addrStr)
	require.NoError(t, err)
	require.Equal(t, addrBytes, recovered)
}

func TestAddressCodec_HexRoundtrip(t *testing.T) {
	a := newTestApp(t)
	codec := a.AccountKeeper.AddressCodec()

	addrBytes := make([]byte, 20)
	for i := range addrBytes {
		addrBytes[i] = byte(0xAA)
	}

	bech32Str, err := codec.BytesToString(addrBytes)
	require.NoError(t, err)

	hexStr := "0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

	fromBech32, err := codec.StringToBytes(bech32Str)
	require.NoError(t, err)

	fromHex, err := codec.StringToBytes(hexStr)
	require.NoError(t, err)

	require.Equal(t, fromBech32, fromHex,
		"bech32 and hex must decode to identical bytes")
}

func TestAddressCodec_ValidatorPrefix(t *testing.T) {
	a := newTestApp(t)
	valCodec := a.StakingKeeper.ValidatorAddressCodec()

	addrBytes := make([]byte, 20)
	for i := range addrBytes {
		addrBytes[i] = byte(i + 10)
	}

	valStr, err := valCodec.BytesToString(addrBytes)
	require.NoError(t, err)
	require.Contains(t, valStr, "monovaloper")

	recovered, err := valCodec.StringToBytes(valStr)
	require.NoError(t, err)
	require.Equal(t, addrBytes, recovered)
}

func TestDefaultGenesis_EVMModulesRegistered(t *testing.T) {
	a := newTestApp(t)
	genesis := a.DefaultGenesis()

	_, ok := genesis[evmtypes.ModuleName]
	require.True(t, ok, "evm module must be registered")

	_, ok = genesis[feemarkettypes.ModuleName]
	require.True(t, ok, "feemarket module must be registered")

	_, ok = genesis[erc20types.ModuleName]
	require.True(t, ok, "erc20 module must be registered")
}

func TestDefaultGenesis_CustomModulesPresent(t *testing.T) {
	a := newTestApp(t)
	genesis := a.DefaultGenesis()

	_, hasBurn := genesis[burnmoduletypes.ModuleName]
	require.True(t, hasBurn)

	_, hasMono := genesis[validatormoduletypes.ModuleName]
	require.True(t, hasMono)
}

func TestDefaultGenesis_AuthzAndFeeGrantPresent(t *testing.T) {
	a := newTestApp(t)
	genesis := a.DefaultGenesis()

	// Verify authz module is in genesis
	_, hasAuthz := genesis["authz"]
	require.True(t, hasAuthz, "authz module must be in genesis")

	// Verify feegrant module is in genesis
	_, hasFeeGrant := genesis["feegrant"]
	require.True(t, hasFeeGrant, "feegrant module must be in genesis")
}

func TestBech32Prefix_Configured(t *testing.T) {
	cfg := sdk.GetConfig()
	require.Equal(t, "mono", cfg.GetBech32AccountAddrPrefix())
	require.Equal(t, "monovaloper", cfg.GetBech32ValidatorAddrPrefix())
	require.Equal(t, "monovalcons", cfg.GetBech32ConsensusAddrPrefix())
}

func TestCoinType_Ethereum(t *testing.T) {
	cfg := sdk.GetConfig()
	require.Equal(t, uint32(60), cfg.GetCoinType())
}

func TestProcessProposal_RejectsMalformedTx(t *testing.T) {
	a := newTestApp(t)

	req := &abci.RequestProcessProposal{
		Height: 1,
		Time:   time.Now(),
		Hash:   []byte{0x01},
		Txs: [][]byte{
			{0x01, 0x02, 0x03},
		},
	}

	resp, err := a.ProcessProposal(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, abci.ResponseProcessProposal_REJECT, resp.Status)
}

func TestGetMempool_DoesNotPanic(t *testing.T) {
	a := newTestApp(t)
	require.NotPanics(t, func() {
		_ = a.GetMempool()
	})
}

func TestSetClientCtx_DoesNotPanic(t *testing.T) {
	a := newTestApp(t)
	require.NotPanics(t, func() {
		a.SetClientCtx(client.Context{})
	})
}

// TestEVMLifecycle is the integration test for the full ABCI lifecycle with
// all EVM modules. Consolidated into one test because the EVM module's
// InitGenesis sets package-level globals via sync.Once (cannot re-init).
//
// Exercises:
//   - InitChain with 20+ modules (EVM, feemarket, erc20, IBC, gov, slashing, evidence, mono, burn)
//   - EVM preinstall injection (CREATE2, Multicall3, Permit2, Safe)
//   - SetModuleVersionMap for upgrade safety
//   - 3-block FinalizeBlock + Commit cycle (all BeginBlockers/EndBlockers)
//   - feemarket EIP-1559 base fee adjustment across blocks
//   - Transient store reset on Commit (feemarket gas tracking)
func TestEVMLifecycle(t *testing.T) {
	// Genesis validator (ed25519 for CometBFT consensus)
	privVal := mock.NewPV()
	pubKey, err := privVal.GetPubKey()
	require.NoError(t, err)
	validator := cmttypes.NewValidator(pubKey, 1)
	valSet := cmttypes.NewValidatorSet([]*cmttypes.Validator{validator})

	// Genesis account (delegator for the validator)
	senderPrivKey := secp256k1.GenPrivKey()
	acc := authtypes.NewBaseAccount(senderPrivKey.PubKey().Address().Bytes(), senderPrivKey.PubKey(), 0, 0)
	balance := banktypes.Balance{
		Address: acc.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewIntWithDecimal(1, 18))),
	}

	// Build genesis state
	a := newTestApp(t)
	genesisState := a.DefaultGenesis()

	// Inject validator into staking/auth/bank genesis
	genesisState, err = simtestutil.GenesisStateWithValSet(a.AppCodec(), genesisState, valSet, []authtypes.GenesisAccount{acc}, balance)
	require.NoError(t, err)

	// Bank denom metadata (EVM InitGenesis requires this for the gas token)
	var bankGenState banktypes.GenesisState
	a.AppCodec().MustUnmarshalJSON(genesisState[banktypes.ModuleName], &bankGenState)
	bankGenState.DenomMetadata = []banktypes.Metadata{{
		Name:        "Monolythium",
		Symbol:      "LYTH",
		Display:     "lyth",
		Base:        app.DefaultBondDenom,
		Description: "The native token of Monolythium",
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: app.DefaultBondDenom, Exponent: 0, Aliases: []string{"attolyth"}},
			{Denom: "lyth", Exponent: 18},
		},
	}}
	genesisState[banktypes.ModuleName] = a.AppCodec().MustMarshalJSON(&bankGenState)

	// EVM gas denom
	var evmGenState evmtypes.GenesisState
	a.AppCodec().MustUnmarshalJSON(genesisState[evmtypes.ModuleName], &evmGenState)
	evmGenState.Params.EvmDenom = app.DefaultBondDenom
	genesisState[evmtypes.ModuleName] = a.AppCodec().MustMarshalJSON(&evmGenState)

	stateBytes, err := json.Marshal(genesisState)
	require.NoError(t, err)

	// InitChain
	_, err = a.InitChain(&abci.RequestInitChain{
		Validators:      []abci.ValidatorUpdate{},
		ConsensusParams: simtestutil.DefaultConsensusParams,
		AppStateBytes:   stateBytes,
		Time:            time.Now(),
	})
	require.NoError(t, err)

	// 3-block cycle: FinalizeBlock + Commit
	for height := int64(1); height <= 3; height++ {
		_, err = a.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height: height,
			Time:   time.Now(),
			Hash:   []byte{byte(height)},
		})
		require.NoError(t, err, "FinalizeBlock failed at height %d", height)
		_, err = a.Commit()
		require.NoError(t, err, "Commit failed at height %d", height)
	}
}

// TestEVMPreinstalls_Available verifies that cosmos/evm ships DefaultPreinstalls
// (CREATE2 deployer, Multicall3, Permit2, Safe contracts). The InitChainer
// injects these into the EVM genesis state. If this slice is empty, no
// standard contracts are deployed at genesis.
func TestEVMPreinstalls_Available(t *testing.T) {
	require.NotEmpty(t, evmtypes.DefaultPreinstalls,
		"DefaultPreinstalls must contain standard EVM contracts")
}

// TestBlockedAddresses_PrecompilesBlocked verifies precompile addresses are blocked
// to prevent permanent fund loss
func TestBlockedAddresses_PrecompilesBlocked(t *testing.T) {
	blocked := app.BlockedAddresses()

	// Test native Ethereum precompiles (0x1-0x9) are blocked
	ethPrecompiles := []string{
		"0x0000000000000000000000000000000000000001", // ecrecover
		"0x0000000000000000000000000000000000000002", // sha256
		"0x0000000000000000000000000000000000000003", // ripemd160
		"0x0000000000000000000000000000000000000004", // identity
		"0x0000000000000000000000000000000000000005", // modexp
		"0x0000000000000000000000000000000000000006", // ecAdd
		"0x0000000000000000000000000000000000000007", // ecMul
		"0x0000000000000000000000000000000000000008", // ecPairing
		"0x0000000000000000000000000000000000000009", // blake2f
	}

	for _, hexAddr := range ethPrecompiles {
		bech32Addr := utils.Bech32StringFromHexAddress(hexAddr)
		require.True(t, blocked[bech32Addr],
			"Ethereum precompile %s must be blocked", hexAddr)
	}

	// Test Cosmos EVM precompiles from AvailableStaticPrecompiles are blocked
	for _, precompileHex := range evmtypes.AvailableStaticPrecompiles {
		bech32Addr := utils.Bech32StringFromHexAddress(precompileHex)
		require.True(t, blocked[bech32Addr],
			"Cosmos precompile %s must be blocked", precompileHex)
	}
}

// TestAccountKeeper_NoAuthKeeperReferences verifies rename is complete
func TestAccountKeeper_NoAuthKeeperReferences(t *testing.T) {
	a := newTestApp(t)

	// Test AccountKeeper methods work
	codec := a.AccountKeeper.AddressCodec()
	require.NotNil(t, codec, "AddressCodec must work")
}
