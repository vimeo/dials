package panels

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/flag"
)

const helpCmdName = "help"

type SetupParams struct {
	NewDials    func(ctx context.Context, defaultCfg interface{}, flagsSource dials.Source) (*dials.Dials, error)
	FlagNameCfg *flag.NameConfig
}

type Something struct {
	Args        []string // everything after the current subcommand (no-flags)
	CommandArgs []string // will include flags (os.Args()). CA[0] = binary name
	Dials       *dials.Dials
	SCPath      []string       // everything until current subcommand, including current SC, no flags
	DialsPath   []*dials.Dials // root command and up to that subcommand
	W           io.Writer
}

type Subpanel interface {
	PanelHelp
	DefaultConfig() interface{}
	SetupParams() SetupParams
	Run(ctx context.Context, s *Something) error
}

type SubCmdHandle struct {
	sp Subpanel
	p  *Panel
}

func (sch *SubCmdHandle) run(ctx context.Context, args []string, s *Something, fCfg *flag.NameConfig) error {
	scmdName := args[0]
	s.SCPath[1] = scmdName

	subFCfg := sch.sp.SetupParams().FlagNameCfg
	if subFCfg == nil {
		subFCfg = fCfg
	}

	fs, nsErr := flag.NewSetWithArgs(subFCfg, sch.sp.DefaultConfig(), args[1:])
	if nsErr != nil {
		return fmt.Errorf("error registering flags: %w", nsErr)
	}

	fs.Flags.SetOutput(s.W)

	ndFunc := func(ctx context.Context, defaultCfg interface{}, flagsSource dials.Source) (*dials.Dials, error) {
		return dials.Config(ctx, defaultCfg, fs)
	}

	if sch.sp.SetupParams().NewDials != nil {
		ndFunc = sch.sp.SetupParams().NewDials
	}

	d, dErr := ndFunc(ctx, sch.sp.DefaultConfig(), fs)
	if dErr != nil {
		return fmt.Errorf("error parsing flags: %w", dErr)
	}

	s.Dials = d
	s.DialsPath[1] = d

	s.Args = fs.Flags.Args()

	if fs.Flags.NArg() == 0 {
		// no subcommands left
		return sch.sp.Run(ctx, s)
	}

	// recurse

	return sch.sp.Run(ctx, s)
}

type Panel struct {
	schMap map[string]*SubCmdHandle
	sp     SetupParams
	dCfg   interface{}
	ph     PanelHelp
	w      io.Writer
}

type PanelHelp interface {
	// scPath is the subcommand-path, including the binary-name (args up to
	// this subcommand with flags stripped out)
	Description(scPath []string) string
	// scPath is the subcommand-path, including the binary-name (args up to
	// this subcommand with flags stripped out)
	ShortUsage(scPath []string) string
	// scPath is the subcommand-path, including the binary-name (args up to
	// this subcommand with flags stripped out)
	LongUsage(scPath []string) string
}

func NewPanel(defaultConfig interface{}, ph PanelHelp, sp SetupParams) Panel {
	return Panel{
		dCfg:   defaultConfig,
		ph:     ph,
		sp:     sp,
		schMap: make(map[string]*SubCmdHandle, 2),
		w:      os.Stdout,
	}
}

func (p *Panel) SetWriter(w io.Writer) {
	p.w = w
}

// Run assumes subcommands are registered before Run is called
func (p *Panel) Run(ctx context.Context, args []string) error {
	fCfg := p.sp.FlagNameCfg
	if fCfg == nil {
		fCfg = flag.DefaultFlagNameConfig()
	}

	w := p.w
	if w == nil {
		w = os.Stdout
	}

	if len(args) < 1 {
		p.w.Write(p.helpString(args[0]))
		return fmt.Errorf("empty argument list")
	}

	argsCopy := args[1:]

	s := Something{
		CommandArgs: args,
		SCPath:      []string{args[0], ""},
		DialsPath:   make([]*dials.Dials, 2),
		W:           w,
	}

	if p.dCfg != nil {
		fs, nsErr := flag.NewSetWithArgs(fCfg, p.dCfg, args[1:])
		if nsErr != nil {
			return fmt.Errorf("error registering flags: %w", nsErr)
		}
		fs.Flags.SetOutput(w)

		var d *dials.Dials

		ndFunc := func(ctx context.Context, defaultCfg interface{}, flagsSource dials.Source) (*dials.Dials, error) {
			return dials.Config(ctx, defaultCfg, fs)
		}

		if p.sp.NewDials != nil {
			ndFunc = p.sp.NewDials
		}

		d, dErr := ndFunc(ctx, p.dCfg, fs)
		if dErr != nil {
			return fmt.Errorf("error parsing flags: %w", dErr)
		}

		s.DialsPath[0] = d
		argsCopy = fs.Flags.Args()
	}

	if len(argsCopy) < 1 {
		p.w.Write(p.helpString(args[0]))
		return fmt.Errorf("no subcommand found")
	}

	scmdName := argsCopy[0]
	sch, ok := p.schMap[scmdName]
	if !ok {

		if scmdName == helpCmdName {
			return p.help(args[0], argsCopy)
		}

		p.w.Write(p.helpString(args[0]))
		return fmt.Errorf("%q subcommand not registered", scmdName)
	}

	return sch.run(ctx, argsCopy, &s, fCfg)
}

func (p *Panel) help(binaryName string, args []string) error {
	if len(args) < 2 {
		p.w.Write(p.helpString(args[0]))
		return nil
	}

	scmdName := args[1]
	sch, ok := p.schMap[scmdName]
	if !ok {
		p.w.Write(p.helpString(args[0]))
		return fmt.Errorf("%q subcommand not registered", scmdName)
	}

	p.w.Write(sch.helpString([]string{binaryName, args[1]}))
	return nil

	// TODO: recursive
}

func (p *Panel) Register(scName string, s Subpanel) (*SubCmdHandle, error) {
	if p.schMap == nil {
		p.schMap = make(map[string]*SubCmdHandle, 1)
	}

	if _, ok := p.schMap[scName]; ok {
		return nil, fmt.Errorf("%q subcommand already registered", scName)
	}

	sch := &SubCmdHandle{
		sp: s,
		p:  p,
	}

	p.schMap[scName] = sch

	return sch, nil
}

func (p *Panel) Process(ctx context.Context) error {
	return p.Run(ctx, os.Args)
}
