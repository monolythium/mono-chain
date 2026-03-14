package keeper_test

import (
	"context"
	"os"
	"testing"

	"cosmossdk.io/core/address"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	evmaddress "github.com/cosmos/evm/encoding/address"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/assert"

	"github.com/monolythium/mono-chain/app"
	"github.com/monolythium/mono-chain/x/mono/keeper"
	module "github.com/monolythium/mono-chain/x/mono/module"
	"github.com/monolythium/mono-chain/x/mono/types"
)

type fixture struct {
	ctx               context.Context
	keeper            keeper.Keeper
	addressCodec      address.Codec
	mockStakingServer *mockStakingMsgServer
	mockBurnServer    *mockBurnMsgServer
}

func initFixture(t *testing.T) *fixture {
	t.Helper()

	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
	require.Equal(t, "mono", sdk.GetConfig().GetBech32AccountAddrPrefix())
	addressCodec := evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	storeService := runtime.NewKVStoreService(storeKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx

	authority := authtypes.NewModuleAddress(types.GovModuleName)
	mockStaking := &mockStakingMsgServer{}
	mockBurn := &mockBurnMsgServer{}
	valAddrCodec := evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())

	k := keeper.NewKeeper(
		storeService,
		encCfg.Codec,
		addressCodec,
		authority,
		nil,
		nil,
		nil,
		mockStaking,
		mockBurn,
		valAddrCodec,
	)

	// Initialize params
	if err := k.Params.Set(ctx, types.DefaultParams()); err != nil {
		assert.NilError(t, err)
	}

	return &fixture{
		ctx:               ctx,
		keeper:            k,
		addressCodec:      addressCodec,
		mockStakingServer: mockStaking,
		mockBurnServer:    mockBurn,
	}
}

func TestMain(m *testing.M) {
	cfg := sdk.GetConfig()
	app.SetBech32Prefixes(cfg)
	app.SetBip44CoinType(cfg)
	os.Exit(m.Run())
}
