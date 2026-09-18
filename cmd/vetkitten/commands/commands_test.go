package commands

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dorcha-inc/vetkitten/internal/forge"
	"github.com/dorcha-inc/vetkitten/internal/report"
	"github.com/dorcha-inc/vetkitten/internal/rules"
)

func TestLoadPolicyFile_MissingFileFallsBackOnlyAtTheDefaultPath(t *testing.T) {
	absent := filepath.Join(t.TempDir(), "absent.yaml")

	p, path, err := loadPolicyFile(absent, false)
	require.NoError(t, err)
	assert.Equal(t, rules.DefaultPreset, p.Preset)
	assert.Equal(t, "", path)

	_, _, err = loadPolicyFile(absent, true)
	assert.Error(t, err)
}

func TestLoadPolicyFromForge_UsesTheRepositoryFileOrTheDefault(t *testing.T) {
	f := &forge.FakeForge{Policies: map[string][]byte{".github/vetkitten.yaml": []byte("preset: ghostty\n")}}
	s := scorer{forge: f}
	ref := forge.Ref{Owner: "o", Repo: "r", Number: 1}

	p, path, err := s.loadPolicyFromForge(context.Background(), ref, "main", ".github/vetkitten.yaml")
	require.NoError(t, err)
	assert.Equal(t, "ghostty", p.Preset)
	assert.Equal(t, ".github/vetkitten.yaml", path)

	p, path, err = s.loadPolicyFromForge(context.Background(), ref, "main", ".github/other.yaml")
	require.NoError(t, err)
	assert.Equal(t, rules.DefaultPreset, p.Preset)
	assert.Equal(t, "", path)
}

func TestRun_CommentsOnNewcomersAndStaysSilentOnCleanReturningAuthors(t *testing.T) {
	f := &forge.FakeForge{}
	f.Add(&forge.PullRequest{Owner: "o", Repo: "r", Number: 1, Title: "clean", Author: forge.Account{Login: "ada", PullRequestsInRepo: 4, MergedPullRequestsInRepo: 3}, Checks: forge.CheckPassing})
	f.Add(&forge.PullRequest{Owner: "o", Repo: "r", Number: 2, Title: "agent", Author: forge.Account{Login: "new"}, HeadBranch: "copilot/x", Checks: forge.CheckPassing})
	policy, err := rules.Preset(rules.DefaultPreset)
	require.NoError(t, err)
	s := scorer{forge: f, policy: policy}
	opts := scoreOptions{format: "markdown", comment: true}

	require.NoError(t, s.run(context.Background(), new(bytes.Buffer), forge.Ref{Owner: "o", Repo: "r", Number: 1}, opts))
	assert.Empty(t, f.Comments["o/r#1"])

	require.NoError(t, s.run(context.Background(), new(bytes.Buffer), forge.Ref{Owner: "o", Repo: "r", Number: 2}, opts))
	require.NoError(t, s.run(context.Background(), new(bytes.Buffer), forge.Ref{Owner: "o", Repo: "r", Number: 2}, opts))
	require.Len(t, f.Comments["o/r#2"], 1)
	assert.Contains(t, f.Comments["o/r#2"][0], report.CommentMarker)
	assert.Contains(t, f.Comments["o/r#2"][0], "Nothing to do before review.")
	assert.Contains(t, f.Comments["o/r#2"][0], "First pull request here from `new`, with no merged pull requests on GitHub yet. The head branch `copilot/x` carries a prefix used by AI coding agents.")
}

func TestAppendSummary_AppendsToTheRunnerFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "summary.md")
	t.Setenv("GITHUB_STEP_SUMMARY", path)
	require.NoError(t, appendSummary("## one\n"))
	require.NoError(t, appendSummary("## two\n"))
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "## one\n## two\n", string(b))
}
