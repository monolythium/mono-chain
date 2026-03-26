package keeper_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/monolythium/mono-chain/app"
	"github.com/monolythium/mono-chain/x/burn/keeper"
	burntestutil "github.com/monolythium/mono-chain/x/burn/testutil"
	"github.com/monolythium/mono-chain/x/burn/types"
)

// failSetStore is a KVStore that returns an error on Set. Reads return
// nil (not found) so collections handles them gracefully via ErrNotFound.
type failSetStore struct{}

func (failSetStore) Get([]byte) ([]byte, error)                                 { return nil, nil }
func (failSetStore) Has([]byte) (bool, error)                                   { return false, nil }
func (failSetStore) Set([]byte, []byte) error                                   { return errors.New("store write failed") }
func (failSetStore) Delete([]byte) error                                        { return nil }
func (failSetStore) Iterator([]byte, []byte) (corestore.Iterator, error)        { return nil, nil }
func (failSetStore) ReverseIterator([]byte, []byte) (corestore.Iterator, error) { return nil, nil }

// failLongBytesToString wraps a real codec but errors on BytesToString for long addresses.
type failLongBytesToString struct{ address.Codec }

func (f failLongBytesToString) BytesToString(bz []byte) (string, error) {
	if len(bz) > 100 {
		return "", errors.New("address too long for codec")
	}
	return f.Codec.BytesToString(bz)
}

type failSetStoreService struct{}

func (failSetStoreService) OpenKVStore(context.Context) corestore.KVStore {
	return failSetStore{}
}

// Force app init() to run, setting sdk.DefaultBondDenom = "alyth".
var _ = app.DefaultBondDenom

type BurnKeeperTestSuite struct {
	suite.Suite
	ctx          sdk.Context
	storeKey     *storetypes.KVStoreKey
	burnKeeper   keeper.Keeper
	bankKeeper   *burntestutil.MockBankKeeper
	msgServer    types.MsgServer
	queryServer  types.QueryServer
	encCfg       moduletestutil.TestEncodingConfig
	authority    string
	addressCodec address.Codec
}

func TestMain(m *testing.M) {
	cfg := sdk.GetConfig()
	app.SetBech32Prefixes(cfg)
	app.SetBip44CoinType(cfg)
	os.Exit(m.Run())
}

func TestBurnKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(BurnKeeperTestSuite))
}

func (s *BurnKeeperTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.encCfg = moduletestutil.MakeTestEncodingConfig()
	s.addressCodec = addresscodec.NewBech32Codec("mono")

	s.storeKey = storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(s.storeKey)
	testCtx := testutil.DefaultContextWithDB(s.T(), s.storeKey, storetypes.NewTransientStoreKey("transient_test"))
	s.ctx = testCtx.Ctx

	s.bankKeeper = burntestutil.NewMockBankKeeper(ctrl)
	authorityAddr := authtypes.NewModuleAddress(types.GovModuleName)
	var err error
	s.authority, err = s.addressCodec.BytesToString(authorityAddr)
	s.Require().NoError(err)

	s.burnKeeper = keeper.NewKeeper(
		storeService,
		s.encCfg.Codec,
		s.addressCodec,
		s.authority,
		s.bankKeeper,
	)

	err = s.burnKeeper.Params.Set(s.ctx, types.DefaultParams())
	s.Require().NoError(err)

	s.msgServer = keeper.NewMsgServerImpl(s.burnKeeper)
	s.queryServer = keeper.NewQueryServerImpl(s.burnKeeper)
}

func (s *BurnKeeperTestSuite) expectBurnCalls(addr sdk.AccAddress, burnAmt int64) []any {
	coins := sdk.NewCoins(sdk.NewCoin("alyth", math.NewInt(burnAmt)))
	return []any{
		s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr, "burn", coins).Return(nil),
		s.bankKeeper.EXPECT().BurnCoins(gomock.Any(), "burn", coins).Return(nil),
	}
}

func (s *BurnKeeperTestSuite) TestBurn_HappyPath() {
	addr := sdk.AccAddress("test_sender_________")
	gomock.InOrder(s.expectBurnCalls(addr, 100)...)

	err := s.burnKeeper.BurnFromAccount(s.ctx, addr, sdk.NewCoin("alyth", math.NewInt(100)))
	s.Require().NoError(err)

	count, err := s.burnKeeper.GlobalBurnCount.Peek(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(uint64(1), count)

	total, err := s.burnKeeper.GlobalBurnTotal.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(math.NewInt(100), total)

	acctTotal, err := s.burnKeeper.AccountBurnTotal.Get(s.ctx, addr)
	s.Require().NoError(err)
	s.Require().Equal(math.NewInt(100), acctTotal)
}

func (s *BurnKeeperTestSuite) TestBurn_WrongDenom() {
	addr := sdk.AccAddress("test_wrong_denom____")
	coins := sdk.NewCoins(sdk.NewCoin("uatom", math.NewInt(100)))
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr, "burn", coins).Return(nil)

	err := s.burnKeeper.BurnFromAccount(s.ctx, addr, sdk.NewCoin("uatom", math.NewInt(100)))
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrInvalidBurnDenom)
}

func (s *BurnKeeperTestSuite) TestBurn_CorruptedCounterPropagatesError() {
	addr := sdk.AccAddress("test_corrupt_ctr____")
	coins := sdk.NewCoins(sdk.NewCoin("alyth", math.NewInt(100)))

	gomock.InOrder(
		s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr, "burn", coins).Return(nil),
		s.bankKeeper.EXPECT().BurnCoins(gomock.Any(), "burn", coins).Return(nil),
	)

	// Write garbage bytes at the Count sequence prefix. decode will fail
	kvStore := s.ctx.KVStore(s.storeKey)
	kvStore.Set(types.GlobalBurnCountPrefix.Bytes(), []byte{0xff, 0xff})

	err := s.burnKeeper.BurnFromAccount(s.ctx, addr, sdk.NewCoin("alyth", math.NewInt(100)))
	s.Require().Error(err)
}

func (s *BurnKeeperTestSuite) TestBurn_TotalSetFailPropagatesError() {
	addr := sdk.AccAddress("test_total_setfail__")
	coins := sdk.NewCoins(sdk.NewCoin("alyth", math.NewInt(100)))

	gomock.InOrder(
		s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr, "burn", coins).Return(nil),
		s.bankKeeper.EXPECT().BurnCoins(gomock.Any(), "burn", coins).Return(nil),
	)

	// Replace GlobalBurnTotal with one backed by a store that errors on Set
	sb := collections.NewSchemaBuilder(failSetStoreService{})
	s.burnKeeper.GlobalBurnTotal = collections.NewItem(sb, types.GlobalBurnTotalPrefix, keeper.GlobalBurnTotalName, sdk.IntValue)
	_, err := sb.Build()
	s.Require().NoError(err)

	err = s.burnKeeper.BurnFromAccount(s.ctx, addr, sdk.NewCoin("alyth", math.NewInt(100)))
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "store write failed")
}

func (s *BurnKeeperTestSuite) TestBurn_GetGlobalTotalFailPropagatesError() {
	addr := sdk.AccAddress("test_total_getfail__")
	coins := sdk.NewCoins(sdk.NewCoin("alyth", math.NewInt(100)))

	gomock.InOrder(
		s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr, "burn", coins).Return(nil),
		s.bankKeeper.EXPECT().BurnCoins(gomock.Any(), "burn", coins).Return(nil),
	)

	kvStore := s.ctx.KVStore(s.storeKey)
	kvStore.Set(types.GlobalBurnTotalPrefix.Bytes(), []byte{0xff, 0xff, 0xff})

	err := s.burnKeeper.BurnFromAccount(s.ctx, addr, sdk.NewCoin("alyth", math.NewInt(100)))
	s.Require().Error(err)
}

func (s *BurnKeeperTestSuite) TestBurn_GetAccountTotalFailPropagatesError() {
	addr := sdk.AccAddress("test_acct_getfail___")
	coins := sdk.NewCoins(sdk.NewCoin("alyth", math.NewInt(100)))

	gomock.InOrder(
		s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr, "burn", coins).Return(nil),
		s.bankKeeper.EXPECT().BurnCoins(gomock.Any(), "burn", coins).Return(nil),
	)

	// Seed then corrupt the account burn total
	err := s.burnKeeper.AccountBurnTotal.Set(s.ctx, addr, math.NewInt(50))
	s.Require().NoError(err)

	kvStore := s.ctx.KVStore(s.storeKey)
	prefix := types.AccountBurnTotalPrefix.Bytes()
	key := make([]byte, 0, len(prefix)+len(addr))
	key = append(key, prefix...)
	key = append(key, addr...)
	kvStore.Set(key, []byte{0xff, 0xff, 0xff})

	err = s.burnKeeper.BurnFromAccount(s.ctx, addr, sdk.NewCoin("alyth", math.NewInt(100)))
	s.Require().Error(err)
}

func (s *BurnKeeperTestSuite) TestBurn_ZeroAmount_Noop() {
	addr := sdk.AccAddress("test_zero_amount____")
	zeroCoin := sdk.NewCoin("alyth", math.ZeroInt())
	// sdk.NewCoins filters out zero coins — bank transfer is a no-op
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(
		gomock.Any(), addr, "burn", sdk.NewCoins(zeroCoin),
	).Return(nil)

	err := s.burnKeeper.BurnFromAccount(s.ctx, addr, zeroCoin)
	s.Require().NoError(err)

	// No trackers updated
	count, err := s.burnKeeper.GlobalBurnCount.Peek(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(uint64(0), count)
}

func (s *BurnKeeperTestSuite) TestBurn_InsufficientFunds() {
	addr := sdk.AccAddress("test_insuf_funds____")
	coins := sdk.NewCoins(sdk.NewCoin("alyth", math.NewInt(200)))
	// Insufficient funds is the bank's responsibility. it rejects the transfer
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr, "burn", coins).
		Return(errors.New("insufficient funds"))

	err := s.burnKeeper.BurnFromAccount(s.ctx, addr, sdk.NewCoin("alyth", math.NewInt(200)))
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "insufficient funds")
}

func (s *BurnKeeperTestSuite) TestBurn_SendCoinsFails() {
	addr := sdk.AccAddress("test_send_fails_____")
	sendErr := errors.New("send coins failed")
	coins := sdk.NewCoins(sdk.NewCoin("alyth", math.NewInt(100)))

	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr, "burn", coins).Return(sendErr)

	err := s.burnKeeper.BurnFromAccount(s.ctx, addr, sdk.NewCoin("alyth", math.NewInt(100)))
	s.Require().Error(err)
	s.Require().ErrorIs(err, sendErr)
}

func (s *BurnKeeperTestSuite) TestBurn_BurnCoinsFails() {
	addr := sdk.AccAddress("test_burn_fails_____")
	burnErr := errors.New("burn coins failed")
	coins := sdk.NewCoins(sdk.NewCoin("alyth", math.NewInt(100)))

	gomock.InOrder(
		s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr, "burn", coins).Return(nil),
		s.bankKeeper.EXPECT().BurnCoins(gomock.Any(), "burn", coins).Return(burnErr),
	)

	err := s.burnKeeper.BurnFromAccount(s.ctx, addr, sdk.NewCoin("alyth", math.NewInt(100)))
	s.Require().Error(err)
	s.Require().ErrorIs(err, burnErr)
}

func (s *BurnKeeperTestSuite) TestBurn_AccumulatesGlobalTotal() {
	addr := sdk.AccAddress("test_accum_global___")

	calls := append(
		s.expectBurnCalls(addr, 100),
		s.expectBurnCalls(addr, 150)...,
	)
	gomock.InOrder(calls...)

	err := s.burnKeeper.BurnFromAccount(s.ctx, addr, sdk.NewCoin("alyth", math.NewInt(100)))
	s.Require().NoError(err)
	err = s.burnKeeper.BurnFromAccount(s.ctx, addr, sdk.NewCoin("alyth", math.NewInt(150)))
	s.Require().NoError(err)

	total, err := s.burnKeeper.GlobalBurnTotal.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(math.NewInt(250), total)

	count, err := s.burnKeeper.GlobalBurnCount.Peek(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(uint64(2), count)
}

func (s *BurnKeeperTestSuite) TestBurn_AccumulatesAccountTotal() {
	addr := sdk.AccAddress("test_accum_acct_____")

	calls := append(
		s.expectBurnCalls(addr, 100),
		s.expectBurnCalls(addr, 75)...,
	)
	gomock.InOrder(calls...)

	err := s.burnKeeper.BurnFromAccount(s.ctx, addr, sdk.NewCoin("alyth", math.NewInt(100)))
	s.Require().NoError(err)
	err = s.burnKeeper.BurnFromAccount(s.ctx, addr, sdk.NewCoin("alyth", math.NewInt(75)))
	s.Require().NoError(err)

	acctTotal, err := s.burnKeeper.AccountBurnTotal.Get(s.ctx, addr)
	s.Require().NoError(err)
	s.Require().Equal(math.NewInt(175), acctTotal)
}

func (s *BurnKeeperTestSuite) TestBurn_SeparateAccountTotals() {
	addr1 := sdk.AccAddress("test_acct_1_________")
	addr2 := sdk.AccAddress("test_acct_2_________")

	calls := append(
		s.expectBurnCalls(addr1, 100),
		s.expectBurnCalls(addr2, 200)...,
	)
	gomock.InOrder(calls...)

	err := s.burnKeeper.BurnFromAccount(s.ctx, addr1, sdk.NewCoin("alyth", math.NewInt(100)))
	s.Require().NoError(err)
	err = s.burnKeeper.BurnFromAccount(s.ctx, addr2, sdk.NewCoin("alyth", math.NewInt(200)))
	s.Require().NoError(err)

	acct1, err := s.burnKeeper.AccountBurnTotal.Get(s.ctx, addr1)
	s.Require().NoError(err)
	s.Require().Equal(math.NewInt(100), acct1)

	acct2, err := s.burnKeeper.AccountBurnTotal.Get(s.ctx, addr2)
	s.Require().NoError(err)
	s.Require().Equal(math.NewInt(200), acct2)
}

func (s *BurnKeeperTestSuite) TestNewKeeper_InvalidAuthorityPanics() {
	storeService := runtime.NewKVStoreService(s.storeKey)
	s.Require().Panics(func() {
		keeper.NewKeeper(storeService, s.encCfg.Codec, s.addressCodec, "bad_authority", s.bankKeeper)
	})
}

func (s *BurnKeeperTestSuite) TestGetAuthority() {
	s.Require().Equal(s.authority, s.burnKeeper.GetAuthority())
}

func (s *BurnKeeperTestSuite) TestGetGlobalBurnTotal_CorruptedStore() {
	kvStore := s.ctx.KVStore(s.storeKey)
	kvStore.Set(types.GlobalBurnTotalPrefix.Bytes(), []byte{0xff, 0xff, 0xff})

	_, err := s.burnKeeper.GetGlobalBurnTotal(s.ctx)
	s.Require().Error(err)
	s.Require().NotErrorIs(err, collections.ErrNotFound)
}

func (s *BurnKeeperTestSuite) TestGetAccountBurnTotal_CorruptedStore() {
	addr := sdk.AccAddress("test_corrupt_acct___")
	err := s.burnKeeper.AccountBurnTotal.Set(s.ctx, addr, math.NewInt(100))
	s.Require().NoError(err)

	// Overwrite the raw value with garbage bytes
	kvStore := s.ctx.KVStore(s.storeKey)
	prefix := types.AccountBurnTotalPrefix.Bytes()
	key := make([]byte, 0, len(prefix)+len(addr))
	key = append(key, prefix...)
	key = append(key, addr...)
	kvStore.Set(key, []byte{0xff, 0xff, 0xff})

	_, err = s.burnKeeper.GetAccountBurnTotal(s.ctx, addr)
	s.Require().Error(err)
	s.Require().NotErrorIs(err, collections.ErrNotFound)
}

func (s *BurnKeeperTestSuite) TestGenesis_RoundTrip() {
	addr1 := sdk.AccAddress("test_gen_acct_1_____")
	addr1Str, err := s.addressCodec.BytesToString(addr1)
	s.Require().NoError(err)

	addr2 := sdk.AccAddress("test_gen_acct_2_____")
	addr2Str, err := s.addressCodec.BytesToString(addr2)
	s.Require().NoError(err)

	genState := types.GenesisState{
		Params: types.NewParams(math.LegacyNewDecWithPrec(5, 1)),
		Count:  7,
		Total:  math.NewInt(500),
		AccountTotals: []types.AccountBurnRecord{
			{Address: addr1Str, Total: math.NewInt(300)},
			{Address: addr2Str, Total: math.NewInt(200)},
		},
	}

	err = s.burnKeeper.InitGenesis(s.ctx, genState)
	s.Require().NoError(err)

	exported, err := s.burnKeeper.ExportGenesis(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(genState.Params, exported.Params)
	s.Require().Equal(genState.Count, exported.Count)
	s.Require().Equal(genState.Total, exported.Total)
	s.Require().Len(exported.AccountTotals, 2)
}

func (s *BurnKeeperTestSuite) TestGenesis_InitInvalidAddressReturnsError() {
	genState := types.GenesisState{
		Params: types.DefaultParams(),
		AccountTotals: []types.AccountBurnRecord{
			{Address: "not_valid_bech32", Total: math.NewInt(100)},
		},
	}

	err := s.burnKeeper.InitGenesis(s.ctx, genState)
	s.Require().Error(err)
}

func (s *BurnKeeperTestSuite) TestGenesis_InitParamsSetFails() {
	sb := collections.NewSchemaBuilder(failSetStoreService{})
	s.burnKeeper.Params = collections.NewItem(sb, types.ParamsKey, "params",
		codec.CollValue[types.Params](s.encCfg.Codec))
	_, _ = sb.Build()

	err := s.burnKeeper.InitGenesis(s.ctx, *types.DefaultGenesisState())
	s.Require().Error(err)
}

func (s *BurnKeeperTestSuite) TestGenesis_InitCountSetFails() {
	sb := collections.NewSchemaBuilder(failSetStoreService{})
	s.burnKeeper.GlobalBurnCount = collections.NewSequence(sb, types.GlobalBurnCountPrefix, keeper.GlobalBurnCountName)
	_, _ = sb.Build()

	err := s.burnKeeper.InitGenesis(s.ctx, *types.DefaultGenesisState())
	s.Require().Error(err)
}

func (s *BurnKeeperTestSuite) TestGenesis_InitTotalSetFails() {
	sb := collections.NewSchemaBuilder(failSetStoreService{})
	s.burnKeeper.GlobalBurnTotal = collections.NewItem(sb, types.GlobalBurnTotalPrefix, keeper.GlobalBurnTotalName, sdk.IntValue)
	_, _ = sb.Build()

	genState := types.DefaultGenesisState()
	genState.Total = math.NewInt(100)

	err := s.burnKeeper.InitGenesis(s.ctx, *genState)
	s.Require().Error(err)
}

func (s *BurnKeeperTestSuite) TestGenesis_InitAccountTotalSetFails() {
	addr := sdk.AccAddress("test_gen_fail_______")
	addrStr, err := s.addressCodec.BytesToString(addr)
	s.Require().NoError(err)

	sb := collections.NewSchemaBuilder(failSetStoreService{})
	s.burnKeeper.AccountBurnTotal = collections.NewMap(sb, types.AccountBurnTotalPrefix, keeper.AccountBurnTotalName, sdk.AccAddressKey, sdk.IntValue)
	_, _ = sb.Build()

	genState := types.DefaultGenesisState()
	genState.AccountTotals = []types.AccountBurnRecord{
		{Address: addrStr, Total: math.NewInt(50)},
	}

	err = s.burnKeeper.InitGenesis(s.ctx, *genState)
	s.Require().Error(err)
}

func (s *BurnKeeperTestSuite) TestGenesis_ExportParamsGetFails() {
	err := s.burnKeeper.Params.Remove(s.ctx)
	s.Require().NoError(err)

	_, err = s.burnKeeper.ExportGenesis(s.ctx)
	s.Require().Error(err)
}

func (s *BurnKeeperTestSuite) TestGenesis_ExportCountPeekFails() {
	kvStore := s.ctx.KVStore(s.storeKey)
	kvStore.Set(types.GlobalBurnCountPrefix.Bytes(), []byte{0xff, 0xff})

	_, err := s.burnKeeper.ExportGenesis(s.ctx)
	s.Require().Error(err)
}

func (s *BurnKeeperTestSuite) TestGenesis_ExportGetGlobalBurnTotalFails() {
	kvStore := s.ctx.KVStore(s.storeKey)
	kvStore.Set(types.GlobalBurnTotalPrefix.Bytes(), []byte{0xff, 0xff, 0xff})

	_, err := s.burnKeeper.ExportGenesis(s.ctx)
	s.Require().Error(err)
}

func (s *BurnKeeperTestSuite) TestGenesis_ExportWalkBytesToStringFails() {
	// Store a long address that the normal codec accepts
	longAddr := sdk.AccAddress(make([]byte, 200))
	err := s.burnKeeper.AccountBurnTotal.Set(s.ctx, longAddr, math.NewInt(1))
	s.Require().NoError(err)

	// Build a keeper with a codec that rejects long addresses in BytesToString
	failCodec := failLongBytesToString{Codec: s.addressCodec}
	storeService := runtime.NewKVStoreService(s.storeKey)
	k := keeper.NewKeeper(storeService, s.encCfg.Codec, failCodec, s.authority, s.bankKeeper)

	_, err = k.ExportGenesis(s.ctx)
	s.Require().Error(err)
}
