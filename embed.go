// Package jstack carries what the binary installs: the skills, the letter, the
// tool list, and the vendor pins. They are embedded at build time so setup runs
// from any directory with no checkout on the machine.
package jstack

import "embed"

//go:embed skills AGENTS.md tools.md vendor.json
var Files embed.FS
