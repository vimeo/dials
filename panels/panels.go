package panels

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/sources/flag"
)

const helpCmdName = "help"

// SetupParams provides ways to configure dials and/or its sources (and possibly other things)
type SetupParams[T any] struct {
	// NewDials lets one override the construction of the dials object.
	// By default, Dials is configured with only a flag source, and no callback registrations.
	// Use of the [dials.Params] type, registering callbacks after construction, using the
	// [github.com/vimeo/dials/ez] package, etc. can all be done from this callback.
	// note: if using the ez package, one must override [ez.Params.FlagSource] with the flagSource argument to this
	// callback.
	NewDials func(ctx context.Context, defaultCfg *T, flagsSource dials.Source) (*dials.Dials[T], error)
	// FlagNameCfg lets one use a non-default flag.NameConfig
	// Defaults to the return value of [flag.DefaultFlagNameConfig]
	FlagNameCfg *flag.NameConfig
}

// BaseHandle is the core portion of Handle, which can be (partially) set by the root command (Panel)
type BaseHandle[RT any] struct {
	Args        []string         // everything after the current subcommand (no-flags)
	CommandArgs []string         // will include flags (os.Args()). CA[0] = binary name
	SCPath      []string         // everything until current subcommand, including current SC, no flags
	RootDials   *dials.Dials[RT] // root command

	W io.Writer
}

// Handle is a parameters type passed to the Run() method of a first-level subcommand
type Handle[RT, T any] struct {
	BaseHandle[RT]
	Dials *dials.Dials[T]
}

// Panel is the basic type for the panels package. It represents the top-level
// command on which subcommands are registered.
// Subcommands are registered using the [Register] function within this package.
type Panel[T any] struct {
	schMap map[string]subCmdRunner[T]
	sp     SetupParams[T]
	dCfg   *T
	ph     PanelHelp
	w      io.Writer
}

// PanelHelp provides help for this (sub)command of various kinds
type PanelHelp interface {
	// Description is an explanation of this (sub)command
	// scPath is the subcommand-path, including the binary-name (args up to
	// this subcommand with flags stripped out)
	Description(scPath []string) string
	// ShortUsage provides information about the usage of this (sub)command
	// in one-ish line.
	// scPath is the subcommand-path, including the binary-name (args up to
	// this subcommand with flags stripped out)
	ShortUsage(scPath []string) string
	// LongUsage provides detailed information about the usage of this
	// (sub)command. (flags will be listed as derived from the flag-set)
	// scPath is the subcommand-path, including the binary-name (args up to
	// this subcommand with flags stripped out)
	LongUsage(scPath []string) string
}

// NewPanel constructs a Panel object
func NewPanel[T any](defaultConfig *T, ph PanelHelp, sp SetupParams[T]) *Panel[T] {
	return &Panel[T]{
		dCfg:   defaultConfig,
		ph:     ph,
		sp:     sp,
		schMap: make(map[string]subCmdRunner[T], 2),
		w:      os.Stdout,
	}
}

// SetWriter overrides the io.Writer to which output is written
func (p *Panel[T]) SetWriter(w io.Writer) {
	p.w = w
}

func (p *Panel[T]) writer() io.Writer {
	if p.w == nil {
		return os.Stdout
	}
	return p.w
}

// Run assumes subcommands are registered before Run is called
func (p *Panel[T]) Run(ctx context.Context, args []string) error {
	fCfg := p.sp.FlagNameCfg
	if fCfg == nil {
		fCfg = flag.DefaultFlagNameConfig()
	}

	w := p.writer()

	if len(args) < 1 {
		w.Write(p.helpString(args[0]))
		return fmt.Errorf("empty argument list")
	}

	argsCopy := args[1:]

	s := BaseHandle[T]{
		CommandArgs: args,
		SCPath:      []string{args[0], ""},
		W:           w,
	}

	if p.dCfg != nil {
		fs, nsErr := flag.NewSetWithArgs(fCfg, p.dCfg, args[1:])
		if nsErr != nil {
			return fmt.Errorf("error registering flags: %w", nsErr)
		}
		fs.Flags.SetOutput(w)

		var d *dials.Dials[T]

		ndFunc := func(ctx context.Context, defaultCfg *T, flagsSource dials.Source) (*dials.Dials[T], error) {
			return dials.Config(ctx, defaultCfg, fs)
		}

		if p.sp.NewDials != nil {
			ndFunc = p.sp.NewDials
		}

		d, dErr := ndFunc(ctx, p.dCfg, fs)
		if dErr != nil {
			return fmt.Errorf("error parsing flags: %w", dErr)
		}

		s.RootDials = d
		argsCopy = fs.Flags.Args()
	}

	if len(argsCopy) < 1 {
		w.Write(p.helpString(args[0]))
		return fmt.Errorf("no subcommand found")
	}

	scmdName := argsCopy[0]
	sch, ok := p.schMap[scmdName]
	if !ok {

		if scmdName == helpCmdName {
			return p.help(args[0], argsCopy)
		}

		w.Write(p.helpString(args[0]))
		return fmt.Errorf("%q subcommand not registered", scmdName)
	}

	return sch.run(ctx, argsCopy, &s, fCfg)
}

func (p *Panel[T]) help(binaryName string, args []string) error {
	w := p.writer()
	if len(args) < 2 {
		w.Write(p.helpString(binaryName))
		return nil
	}

	scmdName := args[1]
	sch, ok := p.schMap[scmdName]
	if !ok {
		w.Write(p.helpString(binaryName))
		return fmt.Errorf("%q subcommand not registered", scmdName)
	}

	w.Write(sch.helpString([]string{binaryName, args[1]}))
	return nil

	// TODO: recursive
}

// Must is a convenience wrapper to panic when an error is encountered.
func Must[T any](sch T, err error) T {
	if err != nil {
		panic(fmt.Errorf("error: failed to make %T: %w", sch, err))
	}
	return sch
}

// Register registers a subcommand (a Subpanel) with a panel.
func Register[RT, T any, SP Subpanel[RT, T]](p *Panel[RT], scName string, s SP) (*SubCmdHandle[RT, T], error) {
	if p.schMap == nil {
		p.schMap = make(map[string]subCmdRunner[RT], 1)
	}

	if _, ok := p.schMap[scName]; ok {
		return nil, fmt.Errorf("%q subcommand already registered", scName)
	}

	sch := &SubCmdHandle[RT, T]{
		sp: s,
		p:  p,
	}

	p.schMap[scName] = sch

	return sch, nil
}

func (p *Panel[T]) Process(ctx context.Context) error {
	return p.Run(ctx, os.Args)
}
