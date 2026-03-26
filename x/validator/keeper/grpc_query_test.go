package keeper_test

import (
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/monolythium/mono-chain/x/validator/keeper"
	"github.com/monolythium/mono-chain/x/validator/types"
)

func (s *ValidatorKeeperTestSuite) TestQueryParams_ReturnsStored() {
	params := types.NewParams(
		sdk.NewCoin("alyth", math.NewInt(500)),
		sdk.NewCoin("alyth", math.NewInt(3000)),
	)
	err := s.validatorKeeper.Params.Set(s.ctx, params)
	s.Require().NoError(err)

	resp, err := s.queryServer.Params(s.ctx, &types.QueryParamsRequest{})
	s.Require().NoError(err)
	s.Require().Equal(params, resp.Params)
}

func (s *ValidatorKeeperTestSuite) TestQueryParams_NoParams() {
	storeKey := storetypes.NewKVStoreKey("empty_query")
	storeService := runtime.NewKVStoreService(storeKey)
	testCtx := testutil.DefaultContextWithDB(s.T(), storeKey, storetypes.NewTransientStoreKey("transient_empty_query"))

	emptyKeeper := keeper.NewKeeper(
		storeService,
		s.encCfg.Codec,
		s.addressCodec,
		s.valAddressCodec,
		s.authority,
		s.burnKeeper,
		s.stakingMsgServer,
	)

	qs := keeper.NewQueryServerImpl(emptyKeeper)
	_, err := qs.Params(testCtx.Ctx, &types.QueryParamsRequest{})
	s.Require().Error(err)
}
