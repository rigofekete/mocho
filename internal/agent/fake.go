package agent

import (
	"context"
	"io"
	"strings"
)

// Call records a single Fake invocation, captured for assertions.
type Call struct {
	Prompt  string
	WorkDir string
	Mode    Mode
	Model   string
}

// Fake is a scripted AgentBackend for tests. It records every Run call and
// returns a reader over the configured canned output. No real process is
// spawned. Set Err to fail the Run before streaming; set EndErr to fail the
// run at EOF (simulating a non-zero exit), mimicking how Opencode surfaces a
// failed synthesis on the final Read.
type Fake struct {
	Output string // canned streamed output
	Err    error  // if set, Run returns this immediately
	EndErr error  // if set, the returned stream fails after outputting Output
	Calls  []Call // each invocation, in order
}

// Run satisfies AgentBackend. It is safe for single-threaded test use.
func (f *Fake) Run(ctx context.Context, prompt, workdir string, mode Mode, model string) (io.ReadCloser, error) {
	f.Calls = append(f.Calls, Call{Prompt: prompt, WorkDir: workdir, Mode: mode, Model: model})
	if f.Err != nil {
		return nil, f.Err
	}
	if f.EndErr != nil {
		return &endErrReader{r: strings.NewReader(f.Output), err: f.EndErr}, nil
	}
	return io.NopCloser(strings.NewReader(f.Output)), nil
}

// Last returns the most recent Call, or a zero Call if never called.
func (f *Fake) Last() Call {
	if len(f.Calls) == 0 {
		return Call{}
	}
	return f.Calls[len(f.Calls)-1]
}

// endErrReader emits the provided output then returns EndErr instead of io.EOF,
// so a scanner observing the stream reports the agent failure.
type endErrReader struct {
	r   *strings.Reader
	err error
}

func (e *endErrReader) Read(p []byte) (int, error) {
	n, err := e.r.Read(p)
	if err == io.EOF && e.err != nil {
		return n, e.err
	}
	return n, err
}

func (e *endErrReader) Close() error { return nil }
