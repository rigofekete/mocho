package agent

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// LightAgentName is the opencode agent profile used to enforce light mode's
// read-only property. It must be defined in the user's opencode config with
// write/edit tools disabled; the backend selects it for ModeLight so the
// restriction is mechanistic, not prompt politeness.
const LightAgentName = "mocho-light"

// Opencode is the default AgentBackend: it spawns `opencode run` against the
// wiki. Tests do not exercise a real opencode process — they substitute Fake.
type Opencode struct{}

// Args returns the `opencode run` argument list for the given run parameters.
// Public so wire-up and tests can inspect the invocation without spawning a
// process. Flag order is not part of the contract; opencode accepts flags in
// any order before the positional message.
func (Opencode) Args(prompt, workdir string, mode Mode, model string) []string {
	return opencodeArgs(prompt, workdir, mode, model)
}

// Run spawns opencode and returns a reader over its streamed stdout. The
// reader surfaces a non-zero process exit as a non-EOF error on the final
// Read, so callers can distinguish synthesis failure from normal EOF.
func (Opencode) Run(ctx context.Context, prompt, workdir string, mode Mode, model string) (io.ReadCloser, error) {
	cmd := exec.CommandContext(ctx, "opencode", opencodeArgs(prompt, workdir, mode, model)...)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("opencode stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start opencode: %w", err)
	}
	return &cmdStream{out: out, cmd: cmd}, nil
}

// opencodeArgs assembles the `opencode run` argument list.
func opencodeArgs(prompt, workdir string, mode Mode, model string) []string {
	args := []string{"run", "--dir", workdir}
	if model != "" {
		args = append(args, "--model", model)
	}
	if mode == ModeLight {
		args = append(args, "--agent", LightAgentName)
	}
	if prompt != "" {
		args = append(args, prompt)
	}
	return args
}

// cmdStream wraps opencode's stdout so reaching EOF also reaps the process and
// reports a non-zero exit as a final Read error. Read must drain stdout to
// EOF before cmd.Wait is called (per exec.Cmd docs); the once guard ensures
// Wait runs exactly once whether reached via Read or Close.
type cmdStream struct {
	out      io.ReadCloser
	cmd      *exec.Cmd
	once     sync.Once
	finalErr error
}

func (c *cmdStream) wait() {
	if werr := c.cmd.Wait(); werr != nil {
		c.finalErr = werr
	}
}

func (c *cmdStream) Read(p []byte) (int, error) {
	n, err := c.out.Read(p)
	if err == io.EOF {
		c.once.Do(c.wait)
		if c.finalErr != nil {
			return n, c.finalErr
		}
	}
	return n, err
}

func (c *cmdStream) Close() error {
	// If the caller never reached EOF (ctx cancel, early abort), kill + reap
	// to avoid leaking a process. Reached via the Run success path read loop.
	c.once.Do(func() {
		if c.cmd.Process != nil {
			_ = c.cmd.Process.Kill()
		}
		c.wait()
	})
	return c.out.Close()
}
