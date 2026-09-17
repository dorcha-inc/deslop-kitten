package rules

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreset_EveryEmbeddedPresetResolves(t *testing.T) {
	names, err := PresetNames()
	require.NoError(t, err)
	require.Contains(t, names, DefaultPreset)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			p, err := Preset(name)
			require.NoError(t, err)
			assert.Equal(t, name, p.Preset)
			assert.NotEmpty(t, p.TrustedBots)
			if name != DefaultPreset {
				assert.NotEmpty(t, p.Source, "a named preset cites the policy it encodes")
			}
		})
	}
}

func TestLoad_OverridesInheritedFieldsAndKeepsTheRest(t *testing.T) {
	p, err := Load(strings.NewReader(`
preset: kubernetes
large_change_files: 10
trusted_bots: ["my-bot[bot]"]
`))
	require.NoError(t, err)
	assert.Equal(t, "kubernetes", p.Preset)
	assert.True(t, p.RequireAIDisclosure)
	assert.True(t, p.ForbidAITrailers)
	assert.Equal(t, 10, p.LargeChangeFiles)
	assert.True(t, p.IsTrustedBot("my-bot[bot]"))
	assert.False(t, p.IsTrustedBot("dependabot[bot]"))
	assert.NotEmpty(t, p.Source)
}

func TestLoad_RejectsBadInput(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "unknown key", in: "requre_linked_issue: true\n", want: "decode policy"},
		{name: "unknown preset", in: "preset: nope\n", want: `unknown preset "nope"`},
		{name: "preset with path", in: "preset: ../x\n", want: "bare word"},
		{name: "negative threshold", in: "large_change_files: -1\n", want: "must be >= 0"},
		{name: "empty list entry", in: "trusted_bots: [' ']\n", want: "empty entry"},
		{name: "bad regex", in: "ai_coauthor_patterns: ['(']\n", want: "ai_coauthor_patterns: compile"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(strings.NewReader(tt.in))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestDefaultPatterns_MatchAgentsAndSpareHumans(t *testing.T) {
	p, err := Preset(DefaultPreset)
	require.NoError(t, err)
	match := func(res []interface{ MatchString(string) bool }, s string) bool {
		for _, re := range res {
			if re.MatchString(s) {
				return true
			}
		}
		return false
	}
	var coauthor, emails []interface{ MatchString(string) bool }
	for _, re := range p.AICoauthorPatterns {
		coauthor = append(coauthor, re)
	}
	for _, re := range p.ModelEmailPatterns {
		emails = append(emails, re)
	}
	assert.True(t, match(coauthor, "Copilot <198982749+Copilot@users.noreply.github.com>"))
	assert.False(t, match(coauthor, "Ada Lovelace <ada@example.com>"))
	assert.True(t, match(emails, "198982749+Copilot@users.noreply.github.com"))
	assert.False(t, match(emails, "12345+ada@users.noreply.github.com"))
}
