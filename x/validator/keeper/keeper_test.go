package keeper_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/monolythium/mono-chain/app"
	"github.com/monolythium/mono-chain/x/validator/keeper"
	valtestutil "github.com/monolythium/mono-chain/x/validator/testutil"
	"github.com/monolythium/mono-chain/x/validator/types"
)

// Force app init() to run, setting sdk.DefaultBondDenom = "alyth".
var _ = app.DefaultBondDenom

// failSetStore is a KVStore that returns an error on Set. Reads return
// nil (not found) so collections handles them gracefully via ErrNotFound.
type failSetStore struct{}

func (failSetStore) Get([]byte) ([]byte, error)                                 { return nil, nil }
func (failSetStore) Has([]byte) (bool, error)                                   { return false, nil }
func (failSetStore) Set([]byte, []byte) error                                   { return errors.New("store write failed") }
func (failSetStore) Delete([]byte) error                                        { return nil }
func (failSetStore) Iterator([]byte, []byte) (corestore.Iterator, error)        { return nil, nil }
func (failSetStore) ReverseIterator([]byte, []byte) (corestore.Iterator, error) { return nil, nil }

type failSetStoreService struct{}

func (failSetStoreService) OpenKVStore(context.Context) corestore.KVStore {
	return failSetStore{}
}

type ValidatorKeeperTestSuite struct {
	suite.Suite
	ctx              sdk.Context
	validatorKeeper  keeper.Keeper
	burnKeeper       *valtestutil.MockBurnKeeper
	stakingMsgServer *valtestutil.MockStakingMsgServer
	msgServer        types.MsgServer
	queryServer      types.QueryServer
	encCfg           moduletestutil.TestEncodingConfig
	authority        string
	addressCodec     address.Codec
	valAddressCodec  address.Codec
}

func TestMain(m *testing.M) {
	cfg := sdk.GetConfig()
	app.SetBech32Prefixes(cfg)
	app.SetBip44CoinType(cfg)
	os.Exit(m.Run())
}

func TestValidatorKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(ValidatorKeeperTestSuite))
}

func (s *ValidatorKeeperTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.encCfg = moduletestutil.MakeTestEncodingConfig()
	s.addressCodec = addresscodec.NewBech32Codec("mono")
	s.valAddressCodec = addresscodec.NewBech32Codec("monovaloper")

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(storeKey)
	testCtx := testutil.DefaultContextWithDB(s.T(), storeKey, storetypes.NewTransientStoreKey("transient_test"))
	s.ctx = testCtx.Ctx

	s.burnKeeper = valtestutil.NewMockBurnKeeper(ctrl)
	s.stakingMsgServer = valtestutil.NewMockStakingMsgServer(ctrl)

	authorityAddr := authtypes.NewModuleAddress(types.GovModuleName)
	var err error
	s.authority, err = s.addressCodec.BytesToString(authorityAddr)
	s.Require().NoError(err)

	s.validatorKeeper = keeper.NewKeeper(
		storeService,
		s.encCfg.Codec,
		s.addressCodec,
		s.valAddressCodec,
		s.authority,
		s.burnKeeper,
		s.stakingMsgServer,
	)

	err = s.validatorKeeper.Params.Set(s.ctx, types.DefaultParams())
	s.Require().NoError(err)

	s.msgServer = keeper.NewMsgServerImpl(s.validatorKeeper)
	s.queryServer = keeper.NewQueryServerImpl(s.validatorKeeper)
}

func (s *ValidatorKeeperTestSuite) TestGenesis_RoundTrip() {
	params := types.NewParams(
		sdk.NewCoin("alyth", math.NewInt(500)),
		sdk.NewCoin("alyth", math.NewInt(2000)),
	)
	err := s.validatorKeeper.Params.Set(s.ctx, params)
	s.Require().NoError(err)

	exported, err := s.validatorKeeper.ExportGenesis(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(params, exported.Params)

	err = s.validatorKeeper.InitGenesis(s.ctx, *exported)
	s.Require().NoError(err)

	got, err := s.validatorKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(params, got)
}

func (s *ValidatorKeeperTestSuite) TestNewKeeper_InvalidAuthority() {
	storeKey := storetypes.NewKVStoreKey("panic_authority")
	storeService := runtime.NewKVStoreService(storeKey)

	s.Require().Panics(func() {
		keeper.NewKeeper(
			storeService,
			s.encCfg.Codec,
			s.addressCodec,
			s.valAddressCodec,
			"not_valid_bech32",
			s.burnKeeper,
			s.stakingMsgServer,
		)
	})
}

func (s *ValidatorKeeperTestSuite) TestGetAuthority() {
	s.Require().Equal(s.authority, s.validatorKeeper.GetAuthority())
}

func (s *ValidatorKeeperTestSuite) TestExportGenesis_NoParams() {
	storeKey := storetypes.NewKVStoreKey("empty_validator")
	storeService := runtime.NewKVStoreService(storeKey)
	testCtx := testutil.DefaultContextWithDB(s.T(), storeKey, storetypes.NewTransientStoreKey("transient_empty"))

	emptyKeeper := keeper.NewKeeper(
		storeService,
		s.encCfg.Codec,
		s.addressCodec,
		s.valAddressCodec,
		s.authority,
		s.burnKeeper,
		s.stakingMsgServer,
	)

	_, err := emptyKeeper.ExportGenesis(testCtx.Ctx)
	s.Require().Error(err)
}
