package file

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/vimeo/dials"
)

// NewSource converts path to an absolute path and returns a source for that file.
func NewSource(path string, decoder dials.Decoder) (*Source, error) {
	absPath, absErr := filepath.Abs(path)
	if absErr != nil {
		return nil, fmt.Errorf("failed to make path %q absolute: %s", path, absErr)
	}
	return &Source{path: absPath, decoder: decoder}, nil
}

// Source is a raw file source.
// Errors reported by the wrapped decoder will be reported wrapped in a
// DecoderErr with the error and file-path populated.
type Source struct {
	path    string
	decoder dials.Decoder
	// We use a random HMAC key for each run since it's not much more
	// expensive and avoids issues with both preimage and collision attacks
	// on our digest-function
	// We can then use this as a comparison to verify whether the contents
	// of the file have changed in the WatchingSource below.
	hmacKey        []byte
	lastHMACSHA256 []byte
	hmacMu         sync.Mutex
}

var _ dials.Source = (*Source)(nil)

func (s *Source) initKey() error {
	s.hmacMu.Lock()
	defer s.hmacMu.Unlock()
	if s.hmacKey != nil {
		return nil
	}

	s.hmacKey = make([]byte, 32)
	_, err := rand.Read(s.hmacKey)
	if err != nil {
		s.hmacKey = nil
		return err
	}
	return nil
}

func (s *Source) hmacReader(r io.Reader) (io.Reader, hash.Hash) {
	if initErr := s.initKey(); initErr != nil {
		// if we fail to initialize the writer, just return the
		// original writer. We'll err on the side of more reloads.
		return r, nil
	}
	h := hmac.New(sha256.New, s.hmacKey)
	out := io.TeeReader(r, h)
	return out, h
}

func (s *Source) lastHMACNew(csum []byte) bool {
	s.hmacMu.Lock()
	defer s.hmacMu.Unlock()
	oldVal := s.lastHMACSHA256
	s.lastHMACSHA256 = csum
	return bytes.Equal(csum, oldVal)
}

type unchangedCSumErr struct {
	csum []byte
}

func (d *unchangedCSumErr) Error() string {
	return fmt.Sprintf("checksum unchanged: %x", d.csum)
}

// DecoderErr wraps another error returned by the inner decoder
type DecoderErr struct {
	Err     error
	Path    string
	Decoder dials.Decoder
}

func (d *DecoderErr) Error() string {
	return fmt.Sprintf("decoder (type %T) error on %q: %s",
		d.Decoder, d.Path, d.Err.Error())
}

func (d *DecoderErr) Unwrap() error {
	return d.Err
}

// Value opens the file and passes it to the Decoder.
func (s *Source) Value(_ context.Context, t *dials.Type) (reflect.Value, error) {
	f, openErr := os.Open(s.path)
	if openErr != nil {
		return reflect.Value{}, openErr
	}
	defer f.Close()

	r, csummer := s.hmacReader(f)
	decoded, decErr := s.decoder.Decode(r, t)
	if decErr != nil {
		return decoded, &DecoderErr{Err: decErr, Path: s.path, Decoder: s.decoder}
	}
	csum := csummer.Sum(nil)
	if s.lastHMACNew(csum) {
		return decoded, &unchangedCSumErr{csum: csum}
	}
	return decoded, nil
}

// WatchOpts contains options, which can be mutated by a WatchOpt
type WatchOpts struct {
	logger       StdLogger
	pollInterval time.Duration
	sigCh        chan os.Signal
}

// WatchOpt functions mutate the state of a WatchOpts, providing optional
// arguments to NewWatchingSource
type WatchOpt func(*WatchOpts)

// WithLogger sets a logger on the new WatchingSource.
func WithLogger(logger StdLogger) WatchOpt {
	return func(o *WatchOpts) {
		o.logger = logger
	}
}

// WithPollInterval configures the new WatchingSource to use a fallback ticker
// to trigger polling for changes to files.
func WithPollInterval(pollInterval time.Duration) WatchOpt {
	return func(o *WatchOpts) {
		o.pollInterval = pollInterval
	}
}

// WithSignalChannel configures the new WatchingSource to use the provided
// channel as a manual trigger for rereading the config file (useful with SIGHUP).
func WithSignalChannel(sigCh chan os.Signal) WatchOpt {
	return func(o *WatchOpts) {
		o.sigCh = sigCh
	}
}

// NewWatchingSource creates a new file watching source that will reload and
// notify if the file is updated.
func NewWatchingSource(
	path string,
	decoder dials.Decoder,
	opts ...WatchOpt,
) (*WatchingSource, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to convert path (%q) to an absolute path: %s",
			path, err)
	}

	o := WatchOpts{}

	for _, opt := range opts {
		opt(&o)
	}

	return &WatchingSource{
		Source: Source{
			path:    absPath,
			decoder: decoder,
		},
		PollInterval: o.pollInterval,
		Reload:       o.sigCh,
		logger:       logWrapper{log: o.logger},
	}, nil
}

// WatchingSource uses fsnotify (inotify, dtrace, etc) to watch for changes to a file
// Errors reported by the wrapped decoder will be reported wrapped in a
// DecoderErr with the error and file-path populated.
type WatchingSource struct {
	Source
	Reload       chan os.Signal
	PollInterval time.Duration
	WG           sync.WaitGroup
	watcher      *fsnotify.Watcher
	logger       logWrapper
}

var _ dials.Source = (*WatchingSource)(nil)
var _ dials.Watcher = (*WatchingSource)(nil)

// Watch Sets up an fsnotify Watcher and starts a background goroutine for watching changes.
func (ws *WatchingSource) Watch(
	ctx context.Context,
	t *dials.Type,
	args dials.WatchArgs) error {
	cleanedPath := filepath.Clean(ws.path)

	// remove one-level of symlinks
	resolvedCfgPath, symlinkErr := filepath.EvalSymlinks(cleanedPath)
	if symlinkErr != nil {
		return fmt.Errorf("failed to follow symlinks from %q: %s",
			cleanedPath, symlinkErr)
	}
	var watchErr error
	ws.watcher, watchErr = fsnotify.NewWatcher()
	if watchErr != nil {
		return fmt.Errorf("failed to initialize watcher: %s", watchErr)
	}

	if addErr := ws.watcher.Add(cleanedPath); addErr != nil {
		return fmt.Errorf("failed to setup watch on file %q: %s",
			cleanedPath, addErr)
	}
	if addErr := ws.watcher.Add(filepath.Dir(cleanedPath)); addErr != nil {
		return fmt.Errorf("failed to setup watch on directory %q: %s",
			cleanedPath, addErr)
	}
	if cleanedPath != resolvedCfgPath {
		if addErr := ws.watcher.Add(filepath.Dir(resolvedCfgPath)); addErr != nil {
			return fmt.Errorf("failed to setup watch on symlink-target dir %q: %s",
				filepath.Dir(resolvedCfgPath), addErr)
		}
	}

	ws.WG.Add(1)
	go ws.watchLoop(ctx, t, cleanedPath, resolvedCfgPath, args)
	return nil
}

// Kubernetes uses its AtomicWriter for updating configmaps, which has
// a somewhat unique structure:
//
//	It creates a timestamped tempdir named by
//	`ioutil.TempDir(w.targetDir, time.Now().UTC().Format("..2006_01_02_15_04_05."))`
//	which contains all the "projected" files, and a `..dir` symlink to that directory.
//	It then creates symlinks from the user-visible location into the
//	`..dir` directory, so it can populate a new timestamped dir and do
//	an atomic rename (from `..data_tmp` to `..data`) of the symlink to
//	make all contents of the configmap atomically updatable.
//	The timestamped directory is then symlinked to `..data_tmp`, which
//	is then atomically renamed over the `..data` directory.
//	At this point, we don't care, but it then cleans up the old directory.
//	doc for the k8s AtomicWriter: https://godoc.org/k8s.io/kubernetes/pkg/volume/util#AtomicWriter
//
// The upshot is that we need to watch the parent directory for
// renames, (which may show up as create/delete pairs) that affect the
// parent directory after one-level of symlink resolution (not
// recursively resolved)
//
// Note: we can be a bit liberal about watching because we verify
// content-changes with an HMAC-SHA256 before reporting anything upstream.
const k8sIntermediateSymlinkDir = "..dir"

func (ws *WatchingSource) watchLoop(
	ctx context.Context,
	t *dials.Type,
	cleanedPath, resolvedCfgPath string,
	args dials.WatchArgs,
) {
	defer ws.WG.Done()
	defer signal.Stop(ws.Reload)
	defer ws.watcher.Close()

	var tickerChan <-chan time.Time
	if ws.PollInterval > 0 {
		ticker := time.NewTicker(ws.PollInterval)
		tickerChan = ticker.C
		defer ticker.Stop()
	}

	watchingFile := true
	eventNumber := 0
	cleanedPathDir := filepath.Dir(cleanedPath)
	cleanedPathDirPlusDir := filepath.Join(cleanedPathDir, k8sIntermediateSymlinkDir)
MAINLOOP:
	for {
		select {
		case <-tickerChan:
		case <-ws.Reload:
		case ev, ok := <-ws.watcher.Events:
			if !ok {
				return
			}
			eventNumber++
			// Filter events down to those pointing at the filename
			// and its parent (both with and without symlinks
			// resolved.
			switch ev.Name {
			case resolvedCfgPath, cleanedPath, cleanedPathDir,
				cleanedPathDirPlusDir, filepath.Dir(resolvedCfgPath):
			default:
				continue MAINLOOP
			}
		case _, ok := <-ws.watcher.Errors:
			if !ok {
				return
			}
			// The only documented error here is an event queue overflow, in which case we missed some events.
			// Fortunately, we can fall-through and get the config itself back into sync.
		case <-ctx.Done():
			return
		}

		newVal, parseErr := ws.Value(ctx, t)

		configExists := !os.IsNotExist(parseErr)
		if !configExists {
			// if the file was renamed or deleted/unlinked,
			// remove its watch.
			if watchingFile {
				if removeErr := ws.watcher.Remove(cleanedPath); removeErr != nil {
					ws.logger.Printf("failed to remove watcher for existing path %q: %s",
						cleanedPath, removeErr)
				}
				watchingFile = false
			}

			// TODO: if the parent directory no longer exists,
			// setup a watch for the first parent that does, so we
			// can re-set the appropriate watches as it gets
			// recreated if necessary.

			// the config doesn't exist; just resume the loop.
			continue
		}
		oldResolvedCfgDir := filepath.Dir(resolvedCfgPath)
		// If the config exists, update the new symlink-path
		if newResolvedPath, symlinkErr := filepath.EvalSymlinks(cleanedPath); symlinkErr == nil {
			resolvedCfgPath = newResolvedPath
		}
		if !watchingFile {
			if addErr := ws.watcher.Add(cleanedPath); addErr != nil {
				ws.logger.Printf("failed to add watcher for path %q: %s",
					cleanedPath, addErr)
			} else {
				watchingFile = true
			}
		}
		ws.updateDirWatches(oldResolvedCfgDir, filepath.Dir(resolvedCfgPath))

		switch t := parseErr.(type) {
		case nil:
			// no error, report upward
			args.ReportNewValue(ctx, newVal)

		case *unchangedCSumErr:
			// Same contents, ignore the new value.
		case *os.SyscallError:
			if !errors.Is(t, os.ErrNotExist) {
				// the file exists, something else failed.
				args.ReportError(ctx, t)
			}
		default:
			args.ReportError(ctx, t)
		}
	}

}

func (ws *WatchingSource) updateDirWatches(oldResolvedCfgDir, resolvedCfgDir string) {
	if oldResolvedCfgDir == resolvedCfgDir {
		return
	}
	// If the config's resolved directory has changed, make sure we
	// remove the old watch after the new one is added so we don't lose change notifications
	if addErr := ws.watcher.Add(resolvedCfgDir); addErr != nil {
		ws.logger.Printf("failed to add new watch for symlink-resolved directory: %q: %s",
			resolvedCfgDir, addErr)
		return
	}
	if removeErr := ws.watcher.Remove(oldResolvedCfgDir); removeErr != nil {
		ws.logger.Printf("failed to remove old watch for old symlink-resolved directory: %q: %s",
			oldResolvedCfgDir, removeErr)
	}
}

// StdLogger is an interface satisified by several logging types, including the
// stdlib `log.Logger`, and `github.com/rs/zerolog.Logger`, and should be trivial
// enough to wrap in other cases.
type StdLogger interface {
	Printf(string, ...any)
	Print(...any)
}

// logWrapper wraps a StdLogger implementation, and gracefully degrades to a
// noop if it's `nil`.
type logWrapper struct {
	log StdLogger
}

func (l *logWrapper) Printf(format string, others ...any) {
	if l.log == nil {
		return
	}
	l.log.Printf(format, others...)
}

func (l *logWrapper) Print(args ...any) {
	if l.log == nil {
		return
	}
	l.log.Print(args...)
}
