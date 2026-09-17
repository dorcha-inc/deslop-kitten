// Package forge reads pull requests from a code forge into one plain
// data model that the signals package evaluates. The GitHub type talks
// to the GitHub REST API. The FakeForge type serves pull requests from
// memory for tests. Both satisfy the Forge interface and both return
// ErrNotFound for a pull request that does not exist.
package forge
