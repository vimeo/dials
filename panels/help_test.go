package panels

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHelpString(t *testing.T) {
	type tlCfg struct{}
	p := Panel[tlCfg]{}

	type srCfg struct{}

	// TODO: add test for panel usage and desc

	tsp := testSubPanel[tlCfg, srCfg]{}

	{
		_, regErr := Register(&p, "foo", &tsp)
		require.NoError(t, regErr)
	}

	{
		_, regErr := Register(&p, "bar", &tsp)
		require.NoError(t, regErr)
	}

	{
		_, regErr := Register(&p, "fizzle", &tsp)
		require.NoError(t, regErr)
	}

	s := p.helpString("fazzle")
	t.Logf("%s", s)

}
