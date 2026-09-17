package rules

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"regexp"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
)

//go:embed presets/*.yaml
var presetFS embed.FS

// DefaultPreset is the preset a policy file inherits when it names none.
const DefaultPreset = "default"

const maxPresetDepth = 8

// File is the on-disk shape of a policy. Every field is optional. A nil
// pointer or nil slice means the preset value stands.
type File struct {
	Preset               string   `yaml:"preset"`
	Source               string   `yaml:"source"`
	RequireLinkedIssue   *bool    `yaml:"require_linked_issue"`
	AcceptedIssueLabels  []string `yaml:"accepted_issue_labels"`
	RequireAIDisclosure  *bool    `yaml:"require_ai_disclosure"`
	ForbidAITrailers     *bool    `yaml:"forbid_ai_trailers"`
	MaxFilesWithoutIssue *int     `yaml:"max_files_without_issue"`
	LargeChangeFiles     *int     `yaml:"large_change_files"`
	BurstEvents          *int     `yaml:"burst_events"`
	TrustedBots          []string `yaml:"trusted_bots"`
	AgentBranchPrefixes  []string `yaml:"agent_branch_prefixes"`
	AICoauthorPatterns   []string `yaml:"ai_coauthor_patterns"`
	ModelEmailPatterns   []string `yaml:"model_email_patterns"`
}

// Policy is a fully resolved, validated policy with compiled patterns.
// Source is the URL of the written policy a preset encodes. A zero
// threshold disables the check it governs. BurstEvents is the number of
// public events inside any three hour window that the comment reports
// as a burst.
type Policy struct {
	Preset               string
	Source               string
	RequireLinkedIssue   bool
	AcceptedIssueLabels  []string
	RequireAIDisclosure  bool
	ForbidAITrailers     bool
	MaxFilesWithoutIssue int
	LargeChangeFiles     int
	BurstEvents          int
	TrustedBots          []string
	AgentBranchPrefixes  []string
	AICoauthorPatterns   []*regexp.Regexp
	ModelEmailPatterns   []*regexp.Regexp
}

// IsTrustedBot reports whether login appears in the trusted bot list.
func (p *Policy) IsTrustedBot(login string) bool {
	return slices.Contains(p.TrustedBots, login)
}

// Preset returns the resolved policy for a named preset.
func Preset(name string) (*Policy, error) {
	f, err := readPreset(name)
	if err != nil {
		return nil, err
	}
	p, err := resolve(f, 0)
	if err != nil {
		return nil, err
	}
	p.Preset = name
	return p, nil
}

// PresetNames returns the embedded preset names in sorted order.
func PresetNames() ([]string, error) {
	entries, err := fs.ReadDir(presetFS, "presets")
	if err != nil {
		return nil, fmt.Errorf("read presets: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, strings.TrimSuffix(e.Name(), ".yaml"))
	}
	slices.Sort(names)
	return names, nil
}

// Load parses a policy file, resolves its preset chain, and validates
// the result. Unknown keys are an error.
func Load(r io.Reader) (*Policy, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read policy: %w", err)
	}
	f, err := decode(data)
	if err != nil {
		return nil, err
	}
	return resolve(f, 0)
}

func decode(data []byte) (*File, error) {
	var f File
	dec := yaml.NewDecoder(bytes.NewReader(data), yaml.Strict())
	if err := dec.Decode(&f); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("decode policy: %w", err)
	}
	return &f, nil
}

func readPreset(name string) (*File, error) {
	if strings.ContainsAny(name, "/\\.") || name == "" {
		return nil, fmt.Errorf("preset name must be a bare word, got %q", name)
	}
	data, err := presetFS.ReadFile("presets/" + name + ".yaml")
	if err != nil {
		return nil, fmt.Errorf("unknown preset %q", name)
	}
	return decode(data)
}

func resolve(f *File, depth int) (*Policy, error) {
	if depth > maxPresetDepth {
		return nil, fmt.Errorf("preset chain deeper than %d", maxPresetDepth)
	}
	var base *Policy
	switch {
	case f.Preset != "":
		parent, err := readPreset(f.Preset)
		if err != nil {
			return nil, err
		}
		base, err = resolve(parent, depth+1)
		if err != nil {
			return nil, err
		}
		base.Preset = f.Preset
	case depth == 0:
		parent, err := readPreset(DefaultPreset)
		if err != nil {
			return nil, err
		}
		base, err = resolve(parent, depth+1)
		if err != nil {
			return nil, err
		}
		base.Preset = DefaultPreset
	default:
		base = &Policy{}
	}
	if err := overlay(base, f); err != nil {
		return nil, err
	}
	if err := base.validate(); err != nil {
		return nil, err
	}
	return base, nil
}

func overlay(p *Policy, f *File) error {
	if f.Source != "" {
		p.Source = f.Source
	}
	if f.RequireLinkedIssue != nil {
		p.RequireLinkedIssue = *f.RequireLinkedIssue
	}
	if f.AcceptedIssueLabels != nil {
		p.AcceptedIssueLabels = slices.Clone(f.AcceptedIssueLabels)
	}
	if f.RequireAIDisclosure != nil {
		p.RequireAIDisclosure = *f.RequireAIDisclosure
	}
	if f.ForbidAITrailers != nil {
		p.ForbidAITrailers = *f.ForbidAITrailers
	}
	if f.MaxFilesWithoutIssue != nil {
		p.MaxFilesWithoutIssue = *f.MaxFilesWithoutIssue
	}
	if f.LargeChangeFiles != nil {
		p.LargeChangeFiles = *f.LargeChangeFiles
	}
	if f.BurstEvents != nil {
		p.BurstEvents = *f.BurstEvents
	}
	if f.TrustedBots != nil {
		p.TrustedBots = slices.Clone(f.TrustedBots)
	}
	if f.AgentBranchPrefixes != nil {
		p.AgentBranchPrefixes = slices.Clone(f.AgentBranchPrefixes)
	}
	var err error
	if f.AICoauthorPatterns != nil {
		if p.AICoauthorPatterns, err = compileAll("ai_coauthor_patterns", f.AICoauthorPatterns); err != nil {
			return err
		}
	}
	if f.ModelEmailPatterns != nil {
		if p.ModelEmailPatterns, err = compileAll("model_email_patterns", f.ModelEmailPatterns); err != nil {
			return err
		}
	}
	return nil
}

func compileAll(field string, patterns []string) ([]*regexp.Regexp, error) {
	out := make([]*regexp.Regexp, 0, len(patterns))
	for _, s := range patterns {
		re, err := regexp.Compile(s)
		if err != nil {
			return nil, fmt.Errorf("%s: compile %q: %w", field, s, err)
		}
		out = append(out, re)
	}
	return out, nil
}

func (p *Policy) validate() error {
	for name, v := range map[string]int{
		"max_files_without_issue": p.MaxFilesWithoutIssue,
		"large_change_files":      p.LargeChangeFiles,
		"burst_events":            p.BurstEvents,
	} {
		if v < 0 {
			return fmt.Errorf("%s must be >= 0, got %d", name, v)
		}
	}
	for name, list := range map[string][]string{
		"agent_branch_prefixes": p.AgentBranchPrefixes,
		"accepted_issue_labels": p.AcceptedIssueLabels,
		"trusted_bots":          p.TrustedBots,
	} {
		if slices.ContainsFunc(list, func(s string) bool { return strings.TrimSpace(s) == "" }) {
			return fmt.Errorf("%s must not contain an empty entry", name)
		}
	}
	return nil
}
