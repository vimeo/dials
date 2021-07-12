package panels

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHelpString(t *testing.T) {
	p := Panel{}

	// TODO: add test for panel usage and desc

	tsp := testSubPanel{}

	{
		_, regErr := p.Register("foo", &tsp)
		require.NoError(t, regErr)
	}

	{
		_, regErr := p.Register("bar", &tsp)
		require.NoError(t, regErr)
	}

	{
		_, regErr := p.Register("fizzle", &tsp)
		require.NoError(t, regErr)
	}

	s := p.helpString("fazzle")
	t.Logf("%s", s)

}
