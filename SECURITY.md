# Security

Report security issues through GitHub's private vulnerability reporting for this repository. Don't open a public issue with exploit details, credentials, or private logs. You should hear back within three business days.

## What this repo is

jstack is a set of markdown skill files plus the `jstack` binary that installs them. The binary embeds the skills, the letter, and `tools.md` at build time. `jstack setup` writes into the skills folder and the instructions file of each harness the human picks, backs up what it overwrites under `~/.jstack/backup/`, and saves the picks in `~/.jstack/config.json`. Nothing here runs a server or handles credentials.

Three things reach the network. Every `jstack setup` run asks GitHub and npm for the latest version of each tool in `tools.md`: one GET per tool to `api.github.com` or `registry.npmjs.org`, five in all, sending only a user agent. If the lookups fail, setup reports the latest as unknown and carries on, so it works offline. The other two are opt-in. `jstack upgrade` downloads the latest GitHub release of this repo, checks the archive against the published `checksums.txt`, runs the new binary's `--version`, and only then replaces the running executable; `install.sh` does the same download and checksum check. `jstack setup` runs the check and install lines from `tools.md` through `sh`. The install lines fetch the tools from their own repos, and setup never runs one without a yes at the prompt, `--install-tools` for a missing tool, or `--update-tools` for an outdated one. Everything runs with the permissions of the person running it.

The skills tell coding agents how to work. A malicious edit to a skill, to `tools.md`, or to `install.sh` could steer an agent or a machine badly, so changes to those go through review and the CLA like anything else, and the release job runs the same checks CI does before it publishes. Read a skill before installing it, the way you'd read a shell script.

Useful reports name the file, the concrete impact, and a short reproduction.
