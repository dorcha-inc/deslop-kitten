// Package rules loads the policy a repository declares for incoming
// pull requests. A policy file names a preset and overrides any field.
// Presets are embedded YAML files under presets/ and encode the written
// contribution policies of projects such as Kubernetes, Ghostty, and
// curl. Load resolves the preset chain, validates every field, and
// compiles every pattern so the signals package works with plain
// values.
package rules
