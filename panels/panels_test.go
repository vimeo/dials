package panels

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/decoders/json"
	"github.com/vimeo/dials/sources/static"
)

type testSubPanel[RT, T any] struct {
	t              testing.TB
	sp             SetupParams[T]
	dCfg           *T
	cmdArgs        []string
	expectedSCPath []string
	expectedArgs   []string
	expSubDCfg     *T
	expPanelDCfg   *RT
}

func (t *testSubPanel[RT, T]) Description(scPath []string) string {
	return "description " + strings.Join(scPath, "-")
}

func (t *testSubPanel[RT, T]) ShortUsage(scPath []string) string {
	return "short " + strings.Join(scPath, "-")
}

func (t *testSubPanel[RT, T]) LongUsage(scPath []string) string {
	return "long " + strings.Join(scPath, "-")
}

func (t *testSubPanel[RT, T]) DefaultConfig() *T {
	return t.dCfg
}

func (t *testSubPanel[RT, T]) SetupParams() SetupParams[T] {
	return t.sp
}

func (t *testSubPanel[RT, T]) Run(ctx context.Context, s *Handle[RT, T]) error {
	t.t.Logf("args: %q, %q, %q", s.Args, s.CommandArgs, s.SCPath)
	assert.Equal(t.t, t.cmdArgs, s.CommandArgs)
	assert.Equal(t.t, t.expectedArgs, s.Args)
	assert.Equal(t.t, t.expectedSCPath, s.SCPath)
	assert.Equal(t.t, t.expSubDCfg, s.Dials.View())

	if t.expPanelDCfg != nil {
		assert.Equal(t.t, t.expPanelDCfg, s.RootDials.View())
	}

	return nil
}

func testPanelsRun[C, SC any](scName string, p *Panel[C], tsp *testSubPanel[C, SC]) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()
		tsp.t = t
		ctx := context.Background()
		sch, regErr := Register(p, scName, tsp)
		require.NoError(t, regErr)
		require.NotNil(t, sch)
		assert.Len(t, p.schMap, 1)

		runErr := p.Run(ctx, tsp.cmdArgs)
		assert.NoError(t, runErr)
	}
}

func TestPanels(t *testing.T) {
	type fibbleFabble struct {
		Fibble string
		Fabble string
	}
	t.Run("multiple_subcommands_with_flags", testPanelsRun("testSubPanel", &Panel[struct{}]{}, &testSubPanel[struct{}, fibbleFabble]{
		t: t,
		dCfg: &fibbleFabble{
			Fibble: "food",
			Fabble: "pizza",
		},
		expSubDCfg: &fibbleFabble{
			Fibble: "bar",
			Fabble: "foo",
		},
		sp:             SetupParams[fibbleFabble]{},
		cmdArgs:        []string{"hello", "testSubPanel", "--fibble=bar", "--fabble=foo", "beThere"},
		expectedSCPath: []string{"hello", "testSubPanel"},
		expectedArgs:   []string{"beThere"},
	}))

	type yellowSub struct {
		Song   string
		Artist string
	}
	t.Run("NewDialsFunc_Panel", testPanelsRun("testSubPanel", &Panel[yellowSub]{
		dCfg: &yellowSub{
			Song:   "yellow submarine",
			Artist: "The Beatles",
		},
		sp: SetupParams[yellowSub]{
			NewDials: func(ctx context.Context, defaultCfg *yellowSub, flagsSource dials.Source) (*dials.Dials[yellowSub], error) {
				return dials.Config(ctx, defaultCfg, flagsSource, &static.StringSource{
					Decoder: &json.Decoder{},
					Data:    `{"Song":"Hello Goodbye"}`,
				})
			},
		},
	}, &testSubPanel[yellowSub, fibbleFabble]{
		dCfg: &fibbleFabble{
			Fibble: "food",
			Fabble: "pizza",
		},
		expSubDCfg: &fibbleFabble{
			Fibble: "bar",
			Fabble: "foo",
		},
		sp:             SetupParams[fibbleFabble]{},
		cmdArgs:        []string{"hello", "testSubPanel", "--fibble=bar", "--fabble=foo", "beThere"},
		expectedSCPath: []string{"hello", "testSubPanel"},
		expectedArgs:   []string{"beThere"},
		expPanelDCfg: &yellowSub{
			Song:   "Hello Goodbye",
			Artist: "The Beatles",
		},
	}))

	t.Run("NewDialsFunc_subpanel", testPanelsRun("testSubPanel", &Panel[struct{}]{}, &testSubPanel[struct{}, fibbleFabble]{
		dCfg: &fibbleFabble{
			Fibble: "food",
			Fabble: "pizza",
		},
		expSubDCfg: &fibbleFabble{
			Fibble: "bar",
			Fabble: "foo",
		},
		sp: SetupParams[fibbleFabble]{
			NewDials: func(ctx context.Context, defaultCfg *fibbleFabble, flagsSource dials.Source) (*dials.Dials[fibbleFabble], error) {
				return dials.Config(ctx, defaultCfg, flagsSource, &static.StringSource{
					Decoder: &json.Decoder{},
					Data:    `{"Fabble":"foo"}`,
				})
			},
		},
		cmdArgs:        []string{"hello", "testSubPanel", "--fibble=bar", "--fabble=foo", "beThere"},
		expectedSCPath: []string{"hello", "testSubPanel"},
		expectedArgs:   []string{"beThere"},
		expPanelDCfg:   nil,
	}))
}
