package panels

import (
	"context"
	"fmt"
	"io"

	"github.com/vimeo/dials/sources/flag"
)

// Subpanel is an interface defining a first-level subcommand.
type Subpanel[RT, T any] interface {
	PanelHelp
	// DefaultConfig returns the default configuration for this subcommand
	DefaultConfig() *T
	// SetupParams returns options describing how to setup Dials (this may expand later)
	SetupParams() SetupParams[T]
	// Run is the main method that's called for this subcommand.
	Run(ctx context.Context, s *Handle[RT, T]) error
}

type subCmdRunner[RT any] interface {
	run(ctx context.Context, args []string, bs *BaseHandle[RT], fCfg *flag.NameConfig) error
	helpString(scPath []string) []byte
	spHelp() PanelHelp

	fs(args []string) (*flag.Set, error)
}

// SubCmdHandle is a handle for a specific subcommand registration, with a
// backpointer to the Panel. (currently an opaque struct)
type SubCmdHandle[RT, T any] struct {
	sp Subpanel[RT, T]
	p  *Panel[RT]
}

var _ subCmdRunner[struct{}] = (*SubCmdHandle[struct{}, struct{}])(nil)

func (sch *SubCmdHandle[RT, T]) spHelp() PanelHelp {
	return sch.sp
}

func (sch *SubCmdHandle[RT, T]) fs(args []string) (*flag.Set, error) {
	subFCfg := sch.sp.SetupParams().flagNameCfg()
	fs, nsErr := flag.NewSetWithArgs(subFCfg, sch.sp.DefaultConfig(), args)
	if nsErr != nil {
		return nil, fmt.Errorf("error registering flags: %w", nsErr)
	}
	return fs, nil
}

func (sch *SubCmdHandle[RT, T]) run(ctx context.Context, args []string, bs *BaseHandle[RT], fCfg *flag.NameConfig) error {
	scmdName := args[0]
	s := Handle[RT, T]{
		BaseHandle: *bs,
	}
	s.SCPath[1] = scmdName

	sp := sch.sp.SetupParams()

	fs, fsErr := sch.fs(args[1:])
	if fsErr != nil {
		return fsErr
	}

	fs.Flags.SetOutput(io.Discard)

	d, dErr := sp.newDials(ctx, sch.sp.DefaultConfig(), fs)
	if dErr != nil {
		return fmt.Errorf("error parsing flags: %w", dErr)
	}

	s.Dials = d

	s.Args = fs.Flags.Args()

	if fs.Flags.NArg() == 0 {
		// no subcommands left
		return sch.sp.Run(ctx, &s)
	}

	// recurse here

	return sch.sp.Run(ctx, &s)
}
