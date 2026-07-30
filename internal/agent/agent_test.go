package agent_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/rigofekete/mocho/internal/agent"
)

func argStrings(a []string) map[string]bool {
	m := make(map[string]bool, len(a))
	for _, s := range a {
		m[s] = true
	}
	return m
}

func TestOpencodeArgsWikiModeWithModel(t *testing.T) {
	args := agent.Opencode{}.Args("synthesize concepts", "/wiki", agent.ModeWiki, "opencode/frontier")
	seen := argStrings(args)
	// Required: --dir, --model, the prompt, "run"; wiki mode never selects a restricted agent.
	if !seen["run"] {
		t.Fatalf("missing run subcommand: %v", args)
	}
	if !seen["--dir"] {
		t.Fatalf("missing --dir: %v", args)
	}
	if !seen["--model"] {
		t.Fatalf("missing --model for wiki mode: %v", args)
	}
	if !seen["opencode/frontier"] {
		t.Fatalf("missing model value: %v", args)
	}
	if seen["--agent"] {
		t.Fatalf("wiki mode must not pass --agent: %v", args)
	}
	if !seen["synthesize concepts"] {
		t.Fatalf("missing prompt: %v", args)
	}
}

func TestOpencodeArgsLightModeSelectsRestrictedAgent(t *testing.T) {
	args := agent.Opencode{}.Args("find what I need", "/wiki", agent.ModeLight, "opencode/cheap")
	seen := argStrings(args)
	if !seen["--model"] {
		t.Fatalf("missing --model in light mode: %v", args)
	}
	if !seen["--agent"] {
		t.Fatalf("light mode must pass --agent: %v", args)
	}
	if !seen[agent.LightAgentName] {
		t.Fatalf("light mode must select restricted agent %q: %v", agent.LightAgentName, args)
	}
}

func TestOpencodeArgsEmptyModelOmitsFlag(t *testing.T) {
	args := agent.Opencode{}.Args("p", "/wiki", agent.ModeWiki, "")
	seen := argStrings(args)
	if seen["--model"] {
		t.Fatalf("empty model must not pass --model: %v", args)
	}
}

func TestFakeRecordsCallAndStreamsOutput(t *testing.T) {
	f := &agent.Fake{Output: "synthesizing...\nwriting concepts/goroutines.md\n"}
	r, err := f.Run(context.Background(), "p", "/w", agent.ModeWiki, "m")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	got, _ := io.ReadAll(r)
	if string(got) != "synthesizing...\nwriting concepts/goroutines.md\n" {
		t.Fatalf("stream = %q", got)
	}
	last := f.Last()
	if last.Prompt != "p" || last.WorkDir != "/w" || last.Mode != agent.ModeWiki || last.Model != "m" {
		t.Fatalf("recorded call = %+v", last)
	}
}

func TestFakeSurfacesError(t *testing.T) {
	f := &agent.Fake{Err: errSentinel{}}
	if _, err := f.Run(context.Background(), "", "", agent.ModeWiki, ""); err != (errSentinel{}) {
		t.Fatalf("err = %v", err)
	}
}

// TestFakeEndErrSurfacesAsFinalReadError verifies the failure-at-EOF behavior
// that the server handler relies on to distinguish a failed synthesis from a
// normal completion.
func TestFakeEndErrSurfacesAsFinalReadError(t *testing.T) {
	f := &agent.Fake{Output: "partial output line\n", EndErr: errSentinel{}}
	r, err := f.Run(context.Background(), "", "", agent.ModeWiki, "")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	got, rerr := io.ReadAll(r)
	if !strings.Contains(string(got), "partial output line") {
		t.Fatalf("expected partial output before error, got %q", got)
	}
	if rerr != (errSentinel{}) {
		t.Fatalf("final read err = %v, want sentinel", rerr)
	}
}

type errSentinel struct{}

func (errSentinel) Error() string { return "sentinel" }
