package panels

// StaticHelp is a [PanelHelp] implementation that returns static strings to
// facilitate usage in the simple-cases where the subcommand name doesn't need
// to be included outside of the already-handled aligned-heading(s).
type StaticHelp struct {
	// Desc is an explanation of this (sub)command's functionality,
	// it will be printed with either Short or Long, so it
	// shouldn't duplicate information that's present in both of those.
	Desc string
	// Short provides information about the usage of this (sub)command
	// in one-ish line.
	// This is printed right after the output of Description, so it should
	// only include commandline args, config, etc. (but be much terser than LongUsage)
	Short string
	// Long provides detailed information about the usage of this
	// (sub)command. (flags will be listed as derived from the flag-set)
	// This is printed right after the output of Description, so it should
	// only include commandline args, config, etc. Unlike Short this
	// should go into as much detail as possible.
	// All flags for this subcommand will be printed, leveraging
	// `dialsdesc` and dialsflag` tags.
	Long string
}

// Description is an explanation of this (sub)command's functionality,
// it will be printed with either ShortUsage or LongUsage, so it
// shouldn't duplicate information that's present in both of those.
// scPath is the subcommand-path, including the binary-name (args up to
// this subcommand with flags stripped out)
func (s StaticHelp) Description(scPath []string) string {
	return s.Desc
}

// ShortUsage provides information about the usage of this (sub)command
// in one-ish line.
// This is printed right after the output of Description, so it should
// only include commandline args, config, etc. (but be much terser than LongUsage)
// scPath is the subcommand-path, including the binary-name (args up to
// this subcommand with flags stripped out)
func (s StaticHelp) ShortUsage(scPath []string) string {
	return s.Short
}

// LongUsage provides detailed information about the usage of this
// (sub)command. (flags will be listed as derived from the flag-set)
// This is printed right after the output of Description, so it should
// only include commandline args, config, etc. Unlike ShortUsage this
// should go into as much detail as possible.
// All flags for this subcommand will be printed, leveraging
// `dialsdesc` and dialsflag` tags.
// scPath is the subcommand-path, including the binary-name (args up to
// this subcommand with flags stripped out)
func (s StaticHelp) LongUsage(scPath []string) string {
	return s.Long
}

var _ PanelHelp = StaticHelp{}
