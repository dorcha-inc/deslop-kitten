package commands

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/jadidbourbaki/vetkitten/internal/disclosure"
	"github.com/jadidbourbaki/vetkitten/internal/forge"
	"github.com/jadidbourbaki/vetkitten/internal/report"
	"github.com/jadidbourbaki/vetkitten/internal/rules"
	"github.com/jadidbourbaki/vetkitten/internal/signals"
)

const defaultPolicyPath = ".github/vetkitten.yaml"

type scoreOptions struct {
	policyPath string
	format     string
	summary    bool
	comment    bool
	timeout    time.Duration
	judgeURL     string
	judgeModel   string
	judgeWait    time.Duration
	judgeTimeout time.Duration
}

// scorer holds the dependencies one run needs. policyPath is empty when
// the default preset applied because no policy file exists.
type scorer struct {
	forge      forge.Forge
	judge      disclosure.Judge
	policy     *rules.Policy
	policyPath string
}

func newScore() *cobra.Command {
	opts := scoreOptions{}
	cmd := &cobra.Command{
		Use:   "score <owner/repo#number | pull request URL>",
		Short: "Check one pull request against the repository's policy and print the report",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.format != "markdown" && opts.format != "json" {
				return fmt.Errorf("format must be markdown or json, got %q", opts.format)
			}
			ref, err := forge.ParseRef(args[0])
			if err != nil {
				return err
			}
			s := scorer{forge: forge.NewGitHub(nil, os.Getenv("GITHUB_TOKEN"))}
			if s.policy, s.policyPath, err = loadPolicy(opts.policyPath, cmd.Flags().Changed("policy")); err != nil {
				return err
			}
			if s.judge, err = newJudge(cmd.Context(), opts); err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), opts.timeout)
			defer cancel()
			return s.run(ctx, cmd.OutOrStdout(), ref, opts)
		},
	}
	cmd.Flags().StringVar(&opts.policyPath, "policy", defaultPolicyPath, "policy file to apply")
	cmd.Flags().StringVar(&opts.format, "format", "markdown", "output format, markdown or json")
	cmd.Flags().BoolVar(&opts.summary, "summary", false, "also append the report to the file named by GITHUB_STEP_SUMMARY")
	cmd.Flags().BoolVar(&opts.comment, "comment", false, "post the report as vetkitten's single comment on the pull request")
	cmd.Flags().DurationVar(&opts.timeout, "timeout", 60*time.Second, "overall deadline for forge and judge requests")
	cmd.Flags().StringVar(&opts.judgeURL, "judge-url", "", "OpenAI-compatible base URL of the disclosure judge, empty skips the disclosure check")
	cmd.Flags().StringVar(&opts.judgeModel, "judge-model", "local", "model name to request from the disclosure judge")
	cmd.Flags().DurationVar(&opts.judgeWait, "judge-wait", 120*time.Second, "how long to wait for the disclosure judge to load its model")
	cmd.Flags().DurationVar(&opts.judgeTimeout, "judge-timeout", 180*time.Second, "deadline for the judge's answer on one pull request")
	return cmd
}

func (s scorer) run(ctx context.Context, out io.Writer, ref forge.Ref, opts scoreOptions) error {
	pr, err := s.forge.PullRequest(ctx, ref.Owner, ref.Repo, ref.Number)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", ref, err)
	}
	findings, err := signals.Run(ctx, signals.Defaults(s.judge), signals.Input{PR: pr, Policy: s.policy})
	if err != nil {
		slog.Default().Warn("score: some signals failed", "ref", ref.String(), "err", err)
	}
	rep := report.Report{Owner: pr.Owner, Repo: pr.Repo, Number: pr.Number, Title: pr.Title, Author: pr.Author, Findings: findings, GeneratedAt: time.Now().UTC()}
	policy := report.Policy{Preset: s.policy.Preset, Source: s.policy.Source, PolicyPath: s.policyPath}

	if opts.comment {
		body := report.AllClear()
		if rep.ShouldComment() {
			body = rep.Comment(policy)
		}
		if err := s.forge.UpsertComment(ctx, pr.Owner, pr.Repo, pr.Number, report.CommentMarker, body, rep.ShouldComment()); err != nil {
			slog.Default().Warn("score: posting the comment failed", "ref", ref.String(), "err", err)
		}
	}
	if opts.summary {
		if err := appendSummary(rep.Markdown(policy)); err != nil {
			return err
		}
	}
	if opts.format == "json" {
		b, err := rep.JSON()
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(out, string(b))
		return err
	}
	_, err = fmt.Fprint(out, rep.Markdown(policy))
	return err
}

// newJudge builds the disclosure judge from the flags and waits for it
// to become ready. An empty judge URL returns a nil judge.
func newJudge(ctx context.Context, opts scoreOptions) (disclosure.Judge, error) {
	if opts.judgeURL == "" {
		return nil, nil
	}
	j, err := disclosure.NewOpenAIJudge(opts.judgeURL, os.Getenv("JUDGE_API_KEY"), opts.judgeModel)
	if err != nil {
		return nil, fmt.Errorf("configure judge: %w", err)
	}
	j.CallTimeout = opts.judgeTimeout
	waitCtx, cancel := context.WithTimeout(ctx, opts.judgeWait)
	defer cancel()
	if err := j.WaitReady(waitCtx); err != nil {
		return nil, err
	}
	return j, nil
}

// loadPolicy reads the policy file and returns the policy with the path
// it came from. A missing file at the default path resolves to the
// default preset with an empty path. A missing file at a path the user
// named is an error.
func loadPolicy(path string, explicit bool) (*rules.Policy, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !explicit {
			p, err := rules.Preset(rules.DefaultPreset)
			return p, "", err
		}
		return nil, "", fmt.Errorf("read policy %s: %w", path, err)
	}
	p, err := rules.Load(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("load policy %s: %w", path, err)
	}
	return p, path, nil
}

func appendSummary(markdown string) (err error) {
	path := os.Getenv("GITHUB_STEP_SUMMARY")
	if path == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return fmt.Errorf("open step summary: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close step summary: %w", cerr)
		}
	}()
	if _, err = io.WriteString(f, markdown); err != nil {
		return fmt.Errorf("write step summary: %w", err)
	}
	return nil
}
