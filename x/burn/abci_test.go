package burn_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppModule_BeginBlock(t *testing.T) {
	f := setupModule(t)
	err := f.keeper.Params.Remove(f.ctx)
	require.NoError(t, err)

	err = f.am.BeginBlock(f.ctx)
	require.Error(t, err)
}
