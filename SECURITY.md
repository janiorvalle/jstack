# Security

Report security issues through GitHub's private vulnerability reporting for this repository. Don't open a public issue with exploit details, credentials, or private logs.

You should hear back within three business days.

## What this repo is

jstack is a set of markdown skill files plus the `jstack` binary that installs them. The binary embeds the skills, the letter, and `tools.md` at build time. `jstack setup` writes into the skills folder and the instructions file of each harness the human picks, backs up what it overwrites under `~/.jstack/backup/`, and saves the picks in `~/.jstack/config.json`. Nothing there runs a server or handles credentials.

Two things do reach the network, and both are opt-in. `jstack upgrade` downloads the latest GitHub release of this repo, checks the archive against the published `checksums.txt`, runs the new binary's `--version`, and only then replaces the running executable. `jstack setup` runs the check and install lines from `tools.md` through `sh`; the install lines fetch the tools from their own repos, and setup never runs one without a yes at the prompt or `--install-tools`. `install.sh` does the same download and checksum check the upgrade does. Everything runs with whatever permissions the person running it has.

The skills tell coding agents how to work. A malicious edit to a skill, to `tools.md`, or to `install.sh` could steer an agent or a machine badly, so changes to those go through review and the CLA like anything else, and the release job runs the same checks CI does before it publishes. Read a skill before installing it, same as you'd read a shell script.

Useful reports name the file, the concrete impact, and a short reproduction.
