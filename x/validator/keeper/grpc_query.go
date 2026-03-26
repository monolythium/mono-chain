package keeper

import (
	"context"

	"github.com/monolythium/mono-chain/x/validator/types"
)

var _ types.QueryServer = queryServer{}

// NewQueryServerImpl returns an implementation of the QueryServer interface.
func NewQueryServerImpl(k Keeper) types.QueryServer {
	return queryServer{k}
}

type queryServer struct {
	k Keeper
}

func (q queryServer) Params(
	ctx context.Context,
	_ *types.QueryParamsRequest,
) (*types.QueryParamsResponse, error) {
	params, err := q.k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{Params: params}, nil
}
