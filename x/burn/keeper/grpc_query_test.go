package keeper_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"

	"github.com/monolythium/mono-chain/x/burn/types"
)

func (s *BurnKeeperTestSuite) TestQueryParams_NilRequest() {
	resp, err := s.queryServer.Params(s.ctx, nil)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
}

func (s *BurnKeeperTestSuite) TestQueryParams_ReturnsStored() {
	params := types.NewParams(math.LegacyNewDecWithPrec(7, 1)) // 0.7
	err := s.burnKeeper.Params.Set(s.ctx, params)
	s.Require().NoError(err)

	resp, err := s.queryServer.Params(s.ctx, &types.QueryParamsRequest{})
	s.Require().NoError(err)
	s.Require().Equal(params, resp.Params)
}

func (s *BurnKeeperTestSuite) TestQueryParams_GetFails() {
	err := s.burnKeeper.Params.Remove(s.ctx)
	s.Require().NoError(err)

	_, err = s.queryServer.Params(s.ctx, &types.QueryParamsRequest{})
	s.Require().Error(err)
}

func (s *BurnKeeperTestSuite) TestQueryBurnStats_PeekFails() {
	kvStore := s.ctx.KVStore(s.storeKey)
	kvStore.Set(types.GlobalBurnCountPrefix.Bytes(), []byte{0xff, 0xff})

	_, err := s.queryServer.BurnStats(s.ctx, &types.QueryBurnStatsRequest{})
	s.Require().Error(err)
}

func (s *BurnKeeperTestSuite) TestQueryBurnStats_Empty() {
	resp, err := s.queryServer.BurnStats(s.ctx, &types.QueryBurnStatsRequest{})
	s.Require().NoError(err)
	s.Require().Equal(uint64(0), resp.TotalBurns)
	// GetTotal() returns math.ZeroInt() when not found
	s.Require().Equal("0", resp.TotalBurned)
}

func (s *BurnKeeperTestSuite) TestQueryBurnStats_AfterBurns() {
	addr := sdk.AccAddress("test_query_stats____")
	gomock.InOrder(s.expectBurnCalls(addr, 100)...)

	err := s.burnKeeper.BurnFromAccount(s.ctx, addr, sdk.NewCoin("alyth", math.NewInt(100)))
	s.Require().NoError(err)

	resp, err := s.queryServer.BurnStats(s.ctx, &types.QueryBurnStatsRequest{})
	s.Require().NoError(err)
	s.Require().Equal(uint64(1), resp.TotalBurns)
	s.Require().Equal("100", resp.TotalBurned)
}

func (s *BurnKeeperTestSuite) TestQueryBurnStats_NilRequest() {
	resp, err := s.queryServer.BurnStats(s.ctx, nil)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
}

func (s *BurnKeeperTestSuite) TestQueryAccountBurns_NoBurns() {
	addr := sdk.AccAddress("test_query_acct_____")
	addrStr, err := s.addressCodec.BytesToString(addr)
	s.Require().NoError(err)

	resp, err := s.queryServer.AccountBurns(s.ctx, &types.QueryAccountBurnsRequest{Address: addrStr})
	s.Require().NoError(err)
	// GetAccountTotal() returns math.ZeroInt() when not found
	s.Require().Equal("0", resp.Amount)
}

func (s *BurnKeeperTestSuite) TestQueryAccountBurns_AfterBurns() {
	addr := sdk.AccAddress("test_query_acct_2___")
	addrStr, err := s.addressCodec.BytesToString(addr)
	s.Require().NoError(err)

	gomock.InOrder(s.expectBurnCalls(addr, 100)...)
	err = s.burnKeeper.BurnFromAccount(s.ctx, addr, sdk.NewCoin("alyth", math.NewInt(100)))
	s.Require().NoError(err)

	resp, err := s.queryServer.AccountBurns(s.ctx, &types.QueryAccountBurnsRequest{Address: addrStr})
	s.Require().NoError(err)
	s.Require().Equal("100", resp.Amount)
}

func (s *BurnKeeperTestSuite) TestQueryAccountBurns_InvalidAddress() {
	_, err := s.queryServer.AccountBurns(s.ctx, &types.QueryAccountBurnsRequest{Address: "bad_addr"})
	s.Require().Error(err)
}

func (s *BurnKeeperTestSuite) TestQueryAccountBurns_NilRequest() {
	_, err := s.queryServer.AccountBurns(s.ctx, nil)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "invalid request")
}

func (s *BurnKeeperTestSuite) TestQueryBurnStats_GetTotalFails() {
	kvStore := s.ctx.KVStore(s.storeKey)
	kvStore.Set(types.GlobalBurnTotalPrefix.Bytes(), []byte{0xff, 0xff, 0xff})

	_, err := s.queryServer.BurnStats(s.ctx, &types.QueryBurnStatsRequest{})
	s.Require().Error(err)
}

func (s *BurnKeeperTestSuite) TestQueryAccountBurns_GetTotalFails() {
	addr := sdk.AccAddress("test_query_corrupt__")
	addrStr, err := s.addressCodec.BytesToString(addr)
	s.Require().NoError(err)

	err = s.burnKeeper.AccountBurnTotal.Set(s.ctx, addr, math.NewInt(100))
	s.Require().NoError(err)

	kvStore := s.ctx.KVStore(s.storeKey)
	prefix := types.AccountBurnTotalPrefix.Bytes()
	key := make([]byte, 0, len(prefix)+len(addr))
	key = append(key, prefix...)
	key = append(key, addr...)
	kvStore.Set(key, []byte{0xff, 0xff, 0xff})

	_, err = s.queryServer.AccountBurns(s.ctx, &types.QueryAccountBurnsRequest{Address: addrStr})
	s.Require().Error(err)
}
