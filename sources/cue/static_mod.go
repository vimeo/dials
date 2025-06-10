package cue

import (
	"context"
	"fmt"
	"reflect"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"

	"github.com/vimeo/dials"
)

// Source parses a directory as a cue module that only processes the cue module once
type Source struct {
	modDir            string
	packageName       string
	fieldPath         *cue.Path
	Tags              []string // cue build tags
	IgnoreNonCueFiles bool
}

// NewSource returns a new [Source].
// the directory is required, but if fieldPath is empty, the entire cue module will be returned as the value.
// packageName is required when using a named queue package. Usually this will be of the form ".:<package_name>"
// ... where the leading "." indicates the subdirectory (relative to dir) to
// read that package's contents from, and <package_name> indicates which
// package to read.
func NewSource(dir, packageName, fieldPath string) (*Source, error) {
	out := Source{
		modDir:            dir,
		packageName:       packageName,
		fieldPath:         nil,
		Tags:              nil,
		IgnoreNonCueFiles: false,
	}

	if fieldPath != "" {
		pth := cue.ParsePath(fieldPath)
		if pth.Err() != nil {
			return nil, fmt.Errorf("invalid field path: %w", pth.Err())
		}
		out.fieldPath = &pth
	}
	return &out, nil
}

// Value provides the current value for the Cue configuration; implementing the [dials.Source] interface.
func (s *Source) Value(ctx context.Context, typ *dials.Type) (reflect.Value, error) {
	cueCfg := load.Config{
		Dir:       s.modDir,
		Tools:     false,
		Tests:     false, // TODO: enable?
		DataFiles: !s.IgnoreNonCueFiles,
	}
	insts := load.Instances([]string{s.packageName}, &cueCfg)

	if len(insts) != 1 {
		return reflect.Value{}, fmt.Errorf("failed to parse config: config yielded non-unity instances: %d", len(insts))
	}
	inst := insts[0]

	if inst.Err != nil {
		return reflect.Value{}, fmt.Errorf("failed to instantiate cue source: %w", inst.Err)
	}

	cctx := cuecontext.New()
	val := cctx.BuildInstance(insts[0])
	if val.Err() != nil {
		return reflect.Value{}, fmt.Errorf("failed to build cue value: %w", val.Err())
	}

	parseVal := val
	if s.fieldPath != nil {
		parseVal = val.LookupPath(*s.fieldPath)
		if parseVal.Err() != nil {
			return reflect.Value{}, fmt.Errorf("failed to lookup path in config: %w", parseVal.Err())
		}
	}

	out := reflect.New(typ.Type())
	if decErr := parseVal.Decode(out.Interface()); decErr != nil {
		return reflect.Value{}, fmt.Errorf("failed to decode config into value: %w", decErr)
	}
	return out, nil
}
