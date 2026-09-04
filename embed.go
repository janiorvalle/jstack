// Package jstack carries what the binary installs: the skills, the letter, the
// tool list, the vendor pins, and the install scripts tools.md lines run. They
// are embedded at build time so setup runs from any directory with no checkout
// on the machine.
package jstack

import "embed"

//go:embed skills AGENTS.md tools.md vendor.json scripts/install-trufflehog.ps1
var Files embed.FS
