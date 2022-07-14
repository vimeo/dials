package panels

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	stdflag "flag"

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

func (s SetupParams[T]) flagNameCfg() *flag.NameConfig {
	if s.FlagNameCfg == nil {
		return flag.DefaultFlagNameConfig()
	}
	return s.FlagNameCfg
}

func (s SetupParams[T]) newDials(ctx context.Context, defaultCfg *T, flagsSource dials.Source) (*dials.Dials[T], error) {
	if s.NewDials == nil {
		return dials.Config(ctx, defaultCfg, flagsSource)
	}

	return s.NewDials(ctx, defaultCfg, flagsSource)
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

func (p *Panel[T]) defCfg() *T {
	if p.dCfg == nil {
		var zCfg T
		return &zCfg
	}
	return p.dCfg
}

// ErrNoSubcommand indicates that there was no subcommand specified
var ErrNoSubcommand = errors.New("no subcommand found")

// ErrPrintHelpSuccess is a sentinel-error that instructs panels to print the help for the selected subcommand, and
// return a nil error/success
var ErrPrintHelpSuccess = errors.New("help requested: success")

// ErrPrintHelpFailure is a sentinel error, which instructs panels to print the help for the selected subcommand, and
// return the full error (note: [errors.Is] just nees to return true)
var ErrPrintHelpFailure = errors.New("help requested: failure")

// Run assumes subcommands are registered before Run is called
func (p *Panel[T]) Run(ctx context.Context, args []string) error {
	fCfg := p.sp.flagNameCfg()

	w := p.writer()

	// This won't happen if we're being executed by a reasonable shell
	if len(args) < 1 {
		w.Write(p.helpString(args[0]))
		return errors.New("empty argument list")
	}

	s := BaseHandle[T]{
		CommandArgs: args,
		SCPath:      []string{args[0], ""},
		W:           w,
	}

	dCfg := p.defCfg()
	fs, nsErr := flag.NewSetWithArgs(fCfg, dCfg, args[1:])
	if nsErr != nil {
		return fmt.Errorf("error registering flags: %w", nsErr)
	}
	rootFlagOutBuf := bytes.Buffer{}
	fs.Flags.SetOutput(&rootFlagOutBuf)
	// If the flagset outlives this function, set it back to using the w writer, so it doesn't pin a random buffer.
	defer fs.Flags.SetOutput(w)

	d, dErr := p.sp.newDials(ctx, dCfg, fs)
	if dErr != nil {
		if errors.Is(dErr, stdflag.ErrHelp) {
			// if one passed `-help` that's not an error, and we want to print the help, just the same as if
			// one passed `help` as the subcommand.
			w.Write(p.helpString(args[0]))
			rootFlagOutBuf.WriteTo(w)
			return nil
		}
		return fmt.Errorf("error parsing flags: %w", dErr)
	}

	s.RootDials = d
	argsCopy := fs.Flags.Args()

	if len(argsCopy) < 1 {
		w.Write(p.helpString(args[0]))
		rootFlagOutBuf.WriteTo(w)
		return ErrNoSubcommand
	}

	scmdName := argsCopy[0]
	sch, ok := p.schMap[scmdName]
	if !ok {

		// since we were able to actually run a subcommand, we know that the root args never dumped the
		// commandline help to the rootFlagOutBuf from an error.
		// call it explicitly
		fs.Flags.PrintDefaults()

		if scmdName == helpCmdName {
			return p.help(rootFlagOutBuf.Bytes(), args[0], argsCopy)
		}

		w.Write(p.helpString(args[0]))
		rootFlagOutBuf.WriteTo(w)
		return fmt.Errorf("%q subcommand not registered", scmdName)
	}

	runErr := sch.run(ctx, argsCopy, &s, fCfg)
	if runErr == nil {
		return nil
	}
	if errors.Is(runErr, stdflag.ErrHelp) || errors.Is(runErr, ErrPrintHelpSuccess) ||
		errors.Is(runErr, ErrPrintHelpFailure) {

		// since we were able to actually run a subcommand, we know that the root args never dumped the
		// commandline help to the rootFlagOutBuf from an error.
		// call it explicitly
		fs.Flags.PrintDefaults()

		p.help(rootFlagOutBuf.Bytes(), args[0], []string{scmdName})

		if errors.Is(runErr, stdflag.ErrHelp) || errors.Is(runErr, ErrPrintHelpSuccess) {
			return nil
		}

	}
	return runErr
}

func (p *Panel[T]) help(flagHelp []byte, binaryName string, scmdPath []string) error {
	w := p.writer()
	if len(scmdPath) < 1 {
		//
		w.Write(p.helpString(binaryName))
		w.Write(flagHelp)
		return nil
	}

	// for now, we only support one level of subcommands
	scmdName := scmdPath[0]
	sch, ok := p.schMap[scmdName]
	if !ok {
		w.Write(p.helpString(binaryName))
		w.Write(flagHelp)
		return fmt.Errorf("%q subcommand not registered", scmdName)
	}

	// This subcommand exists
	w.Write(sch.helpString([]string{binaryName, scmdName}))

	// don't bother passing the right arg-list (we're passing an empty set anyway)
	scFS, fsErr := sch.fs([]string{})
	if fsErr != nil {
		return fmt.Errorf("error registering flags: %w", fsErr)
	}
	scFlagBuf := bytes.Buffer{}
	scFS.Flags.SetOutput(&scFlagBuf)
	scFS.Flags.PrintDefaults()

	// dump the subcommand flags:
	fmt.Fprintf(p.w, "%s flags:\n", scmdName)
	scFlagBuf.WriteTo(p.w)

	// dump the root flags
	fmt.Fprintf(p.w, "Root Flags:\n")
	w.Write(flagHelp)
	return nil

	// TODO: iterate for inner subcommands
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
