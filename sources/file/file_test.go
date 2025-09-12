package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/decoders/json"
)

type testStdLogger struct {
	t testing.TB
}

func (t *testStdLogger) Printf(format string, others ...any) {
	t.t.Helper()
	t.t.Logf(format, others...)
}

func (t *testStdLogger) Print(args ...any) {
	t.t.Helper()
	t.t.Log(args...)
}

type config struct {
	SecretOfLife int
	NumBeatles   int
}

func tmpDir(t testing.TB) string {
	t.Helper()
	dir, dirErr := os.MkdirTemp("", "dials_file")
	require.NoError(t, dirErr, "failed to create temporary directory")
	return dir
}

func TestWatchingFile(t *testing.T) {
	t.Parallel()

	dir := tmpDir(t)
	defer os.RemoveAll(dir)

	firstConfig := writeTestConfig(t, dir, `{
        "secretOfLife": 42,
        "numBeatles": 4
    }`)
	defer os.Remove(firstConfig)

	secondConfig := writeTestConfig(t, dir, `{
        "secretOfLife": 47,
        "numBeatles": 4
    }`)
	defer os.Remove(secondConfig)

	myConfig := &config{}

	watchingFile, watchingErr := NewWatchingSource(firstConfig, &json.Decoder{}, WithLogger(&testStdLogger{t}))
	require.NoError(t, watchingErr, "construction failure")
	defer watchingFile.WG.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d, err := dials.Config(ctx, myConfig, watchingFile)
	assert.NoError(t, err)

	c := d.View()
	assert.Equal(t, 42, c.SecretOfLife)
	assert.Equal(t, 4, c.NumBeatles)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		<-d.Events()
		wg.Done()
	}()

	// rename the second file over the top of the first one
	assert.NoError(t, os.Rename(secondConfig, firstConfig))

	wg.Wait()

	c = d.View()
	assert.Equal(t, 47, c.SecretOfLife)
	assert.Equal(t, 4, c.NumBeatles)
}

func TestWatchingFileWithRelativePathAndChdir(t *testing.T) {
	initWD, wdErr := os.Getwd()
	require.NoError(t, wdErr, "get working directory")
	defer os.Chdir(initWD)
	t.Parallel()

	dir := tmpDir(t)
	defer os.RemoveAll(dir)

	firstConfig := writeTestConfig(t, dir, `{
        "secretOfLife": 42,
        "numBeatles": 4
    }`)
	defer os.Remove(firstConfig)

	secondConfig := writeTestConfig(t, dir, `{
        "secretOfLife": 47,
        "numBeatles": 4
    }`)
	defer os.Remove(secondConfig)

	myConfig := &config{}

	require.NoErrorf(t, os.Chdir(dir), "chdir to %s", dir)

	relFname, relErr := filepath.Rel(dir, firstConfig)
	require.NoErrorf(t, relErr, "failed to construct relative path from dir %q to firstconfig %q", dir, firstConfig)

	watchingFile, watchingErr := NewWatchingSource(relFname, &json.Decoder{}, WithLogger(&testStdLogger{t}))
	require.NoError(t, watchingErr, "construction failure")
	defer watchingFile.WG.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d, err := dials.Config(ctx, myConfig, watchingFile)
	assert.NoError(t, err)

	c := d.View()
	assert.Equal(t, 42, c.SecretOfLife)
	assert.Equal(t, 4, c.NumBeatles)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		<-d.Events()
		wg.Done()
	}()

	// before we rename over the old file, move to another directory (the
	// original directory seems like a good place to leave ourselves)
	assert.NoError(t, os.Chdir(initWD))

	// rename the second file over the top of the first one
	assert.NoError(t, os.Rename(secondConfig, firstConfig))

	wg.Wait()

	c = d.View()
	assert.Equal(t, 47, c.SecretOfLife)
	assert.Equal(t, 4, c.NumBeatles)
}

func TestWatchingFileWithRemove(t *testing.T) {
	t.Parallel()

	dir := tmpDir(t)
	defer os.RemoveAll(dir)

	firstConfig := writeTestConfig(t, dir, `{
        "secretOfLife": 42,
        "numBeatles": 4
    }`)
	defer os.Remove(firstConfig)

	myConfig := &config{}

	watchingFile, watchingErr := NewWatchingSource(firstConfig, &json.Decoder{}, WithLogger(&testStdLogger{t}))
	require.NoError(t, watchingErr, "construction failure")
	defer watchingFile.WG.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d, err := dials.Config(ctx, myConfig, watchingFile)
	require.NoErrorf(t, err, "failed to construct watcher (on file %q)", firstConfig)

	c := d.View()
	assert.Equal(t, 42, c.SecretOfLife)
	assert.Equal(t, 4, c.NumBeatles)

	watcherCtx, cancel := context.WithCancel(context.Background())
	completed := make(chan struct{})
	go func(ctx context.Context, completed chan struct{}) {
		select {
		case <-d.Events():
		case <-ctx.Done():
		}
		close(completed)
	}(watcherCtx, completed)

	assert.NoError(t, os.Remove(firstConfig))
	timer := time.NewTimer(1 * time.Second)

	select {
	case <-completed:
		t.Errorf("erroneously received event on deleted file")
	case <-timer.C:
	}
	cancel()

	// should still be set to what it was before
	c = d.View()
	assert.Equal(t, 42, c.SecretOfLife)
	assert.Equal(t, 4, c.NumBeatles)
}

func TestWatchingFileWithTrickle(t *testing.T) {
	t.Parallel()

	dir := tmpDir(t)
	defer os.RemoveAll(dir)

	firstConfig := writeTestConfig(t, dir, `{
        "secretOfLife": 42,
        "numBeatles": 4
    }`)
	defer os.Remove(firstConfig)

	myConfig := &config{}

	watchingFile, watchingErr := NewWatchingSource(firstConfig, &json.Decoder{}, WithLogger(&testStdLogger{t}))
	require.NoError(t, watchingErr, "construction failure")
	defer watchingFile.WG.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d, err := dials.Config(ctx, myConfig, watchingFile)
	assert.NoError(t, err)

	c := d.View()
	assert.Equal(t, 42, c.SecretOfLife)
	assert.Equal(t, 4, c.NumBeatles)

	watcherCtx, cancel := context.WithCancel(context.Background())
	completed := make(chan struct{})
	counter := int32(0)
	go func(ctx context.Context, completed chan struct{}) {
	LOOP:
		for {
			select {
			case <-d.Events():
				atomic.AddInt32(&counter, 1)
			case <-ctx.Done():
				break LOOP
			}
		}
		close(completed)
	}(watcherCtx, completed)

	// open and truncate the config file
	f, err := os.OpenFile(firstConfig, os.O_RDWR|os.O_TRUNC, 0640)

	assert.NoError(t, err)

	newContents := `{
        "secretOfLife": 11,
        "numBeatles": 4
    }`
	// write the new configuration annoyingly one character at a time
	// on purpose so we can make sure that the configuration isn't flapping
	// with erroneous updates even though parsing fails.
	for _, char := range newContents {
		// we don't care about utf-8 here...
		f.Write([]byte{byte(char)})
		f.Sync()
	}
	f.Close()
	time.Sleep(2 * time.Second)
	cancel()

	<-completed

	// We should be suppressing spurious notifications
	assert.EqualValues(t, atomic.LoadInt32(&counter), 1)

	// should still be set to what it was before
	c = d.View()
	assert.Equal(t, 11, c.SecretOfLife)
	assert.Equal(t, 4, c.NumBeatles)
}

func TestWatchingFileWithK8SEmulatedAtomicWriter(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("k8s config updates are more complicated on windows skipping for now.")
	}
	t.Parallel()

	const fname = "fimbat.json"

	wdir, tmpdirErr := os.MkdirTemp("", "dials_file_test-")
	require.NoError(t, tmpdirErr, "create tmpDir")
	defer os.RemoveAll(wdir)

	configPath := filepath.Join(wdir, fname)

	const subdirPath = "..dir"
	const subdirTmpPath = "..dir_tmp"

	const firstTSDir = "..timestamped_dir-1."
	symlinkPath := filepath.Join(wdir, subdirPath)
	firstRealContentsDir := filepath.Join(wdir, firstTSDir)
	firstConfigPath := filepath.Join(firstRealContentsDir, fname)

	require.NoErrorf(t, os.Mkdir(firstRealContentsDir, 0755), "failed to create second tsdir %q", firstRealContentsDir)
	require.NoErrorf(t, os.Symlink(firstTSDir, symlinkPath), "failed to create symlink from %q to %q", firstTSDir, symlinkPath)

	intermediateSymlinkPath := filepath.Join(subdirPath, fname)
	require.NoErrorf(t, os.Symlink(intermediateSymlinkPath, configPath), "failed to create symlink from %q to %q",
		intermediateSymlinkPath,
		configPath)

	require.NoError(t, os.WriteFile(firstConfigPath, []byte(`{
        "secretOfLife": 42,
        "numBeatles": 4
    }`), 0400), "failed to write contents of first version of config file")

	myConfig := &config{}

	watchingFile, watchingErr := NewWatchingSource(configPath, &json.Decoder{}, WithLogger(&testStdLogger{t}))
	require.NoError(t, watchingErr, "construction failure")
	defer watchingFile.WG.Wait()

	ctx, outerCancel := context.WithCancel(context.Background())
	defer outerCancel()
	d, err := dials.Config(ctx, myConfig, watchingFile)
	assert.NoError(t, err)

	c := d.View()
	assert.Equal(t, 42, c.SecretOfLife)
	assert.Equal(t, 4, c.NumBeatles)

	watcherCtx, cancel := context.WithCancel(context.Background())
	completed := make(chan struct{})
	received := make(chan struct{})
	counter := int32(0)
	go func(ctx context.Context, completed chan struct{}) {
		expectedSecret := 11
	LOOP:
		for {
			select {
			case conf := <-d.Events():
				atomic.AddInt32(&counter, 1)
				assert.EqualValues(t, 4, conf.NumBeatles)
				assert.EqualValues(t, expectedSecret, conf.SecretOfLife)
				expectedSecret++
				received <- struct{}{}
			case <-ctx.Done():
				break LOOP
			}
		}
		close(completed)
	}(watcherCtx, completed)

	// Update the config 13 times so we have confidence that this works
	// repeatably with the k8s code.
	for i := 2; i < 15; i++ {
		nextTSDir := fmt.Sprintf("..timestamped_dir-%d.", i)
		nextRealContentsDir := filepath.Join(wdir, nextTSDir)
		require.NoErrorf(t, os.Mkdir(nextRealContentsDir, 0755), "failed to create second tsdir %q", nextRealContentsDir)
		fullSubdirTmpPath := filepath.Join(wdir, subdirTmpPath)
		require.NoErrorf(t, os.Symlink(nextTSDir, fullSubdirTmpPath), "failed to create symlink from %q to %q",
			nextTSDir, fullSubdirTmpPath)

		secondRealContentsPath := filepath.Join(nextRealContentsDir, fname)
		require.NoError(t, os.WriteFile(secondRealContentsPath, fmt.Appendf(nil, `{
        "secretOfLife": %d,
        "numBeatles": 4
    }`, 9+i), 0400),
			"failed to write new config")

		require.NoErrorf(t, os.Rename(fullSubdirTmpPath, symlinkPath), "failed to rename from %q to %q", fullSubdirTmpPath, symlinkPath)
		<-received
	}

	cancel()
	<-completed

	// We should be suppressing spurious notifications, so we should only
	// see 13 distinct value events.
	assert.EqualValues(t, atomic.LoadInt32(&counter), 13)

	// let it panic if it isn't the &config type we expect
	c = d.View()
	assert.Equal(t, 9+14, c.SecretOfLife)
	assert.Equal(t, 4, c.NumBeatles)
}

const watchingFilePattern = "watching-file"

func writeTestConfig(t testing.TB, dir, data string) string {
	t.Helper()
	f, err := os.CreateTemp(dir, watchingFilePattern)
	assert.NoError(t, err)
	defer f.Close()
	f.WriteString(data)
	return f.Name()
}
