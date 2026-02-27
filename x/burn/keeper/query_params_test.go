package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/monolythium/mono-chain/x/burn/keeper"
	"github.com/monolythium/mono-chain/x/burn/types"
)

func TestParamsQuery(t *testing.T) {
	f := initFixture(t)

	qs := keeper.NewQueryServerImpl(f.keeper)
	params := types.DefaultParams()
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	response, err := qs.Params(f.ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}

func TestParamsQuery_NilRequest(t *testing.T) {
	f := initFixture(t)

	qs := keeper.NewQueryServerImpl(f.keeper)

	_, err := qs.Params(f.ctx, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid request")
}

// TestQuery_BurnStats verifies the BurnStats query returns correct totals
func TestQuery_BurnStats(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	// Initial state - no burns
	resp, err := qs.BurnStats(f.ctx, &types.QueryBurnStatsRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, uint64(0), resp.TotalBurns)
	require.Equal(t, "<nil>", resp.TotalBurned)

	// Simulate burns by updating the collections directly
	// Increment burn count 3 times
	for i := 0; i < 3; i++ {
		_, err = f.keeper.BurnCount.Next(f.ctx)
		require.NoError(t, err)
	}

	// Set burn total to 500 alyth
	burnTotal := sdk.NewCoin("alyth", math.NewInt(500))
	err = f.keeper.BurnTotal.Set(f.ctx, burnTotal)
	require.NoError(t, err)

	// Query again
	resp, err = qs.BurnStats(f.ctx, &types.QueryBurnStatsRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, uint64(3), resp.TotalBurns)
	require.Equal(t, "500alyth", resp.TotalBurned)
}

// TestQuery_BurnStats_NilRequest verifies nil request is handled
func TestQuery_BurnStats_NilRequest(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	_, err := qs.BurnStats(f.ctx, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid request")
}

// TestQuery_AccountBurns verifies per-account burn totals
func TestQuery_AccountBurns(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	addr := sdk.AccAddress("test_account_query")

	// Query non-existent account - should return empty
	resp, err := qs.AccountBurns(f.ctx, &types.QueryAccountBurnsRequest{
		Address: addr.String(),
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, "<nil>", resp.Amount)

	// Set account burn total
	accountTotal := sdk.NewCoin("alyth", math.NewInt(250))
	err = f.keeper.BurnAccountTotal.Set(f.ctx, addr, accountTotal)
	require.NoError(t, err)

	// Query again
	resp, err = qs.AccountBurns(f.ctx, &types.QueryAccountBurnsRequest{
		Address: addr.String(),
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, "250alyth", resp.Amount)
}

// TestQuery_AccountBurns_InvalidAddress verifies invalid address handling
func TestQuery_AccountBurns_InvalidAddress(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	// Invalid bech32 address
	resp, err := qs.AccountBurns(f.ctx, &types.QueryAccountBurnsRequest{
		Address: "not_a_valid_address",
	})
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "invalid address")
}

// TestQuery_AccountBurns_NilRequest verifies nil request is handled
func TestQuery_AccountBurns_NilRequest(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	_, err := qs.AccountBurns(f.ctx, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid request")
}

// TestQuery_BurnStats_WithErrors tests error scenarios in BurnStats
func TestQuery_BurnStats_WithErrors(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	// Test with valid request - should not error even with no data
	resp, err := qs.BurnStats(f.ctx, &types.QueryBurnStatsRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Simulate burns to test Has() and Get() paths
	_, err = f.keeper.BurnCount.Next(f.ctx)
	require.NoError(t, err)

	err = f.keeper.BurnTotal.Set(f.ctx, sdk.NewCoin("alyth", math.NewInt(1000)))
	require.NoError(t, err)

	// Query with data present
	resp, err = qs.BurnStats(f.ctx, &types.QueryBurnStatsRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(1), resp.TotalBurns)
	require.Equal(t, "1000alyth", resp.TotalBurned)
}

// TestQuery_AccountBurns_WithData tests AccountBurns with actual data
func TestQuery_AccountBurns_WithData(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	addr := sdk.AccAddress("test_account_data")

	// Set account burn data
	err := f.keeper.BurnAccountTotal.Set(f.ctx, addr, sdk.NewCoin("alyth", math.NewInt(5000)))
	require.NoError(t, err)

	// Query with data present
	resp, err := qs.AccountBurns(f.ctx, &types.QueryAccountBurnsRequest{
		Address: addr.String(),
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, "5000alyth", resp.Amount)

	// Query different address with no data
	addr2 := sdk.AccAddress("no_data_account")
	resp, err = qs.AccountBurns(f.ctx, &types.QueryAccountBurnsRequest{
		Address: addr2.String(),
	})
	require.NoError(t, err)
	require.Equal(t, "<nil>", resp.Amount)
}
