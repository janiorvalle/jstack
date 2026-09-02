# Security

Report security issues through GitHub's private vulnerability reporting for this repository. Don't open a public issue with exploit details, credentials, or private logs.

You should hear back within three business days.

## What this repo is

jstack is a set of markdown skill files and two small python scripts. Nothing here runs a server, makes network calls, or handles credentials. The `sync` script copies folders into a local skills directory. The `worktree_lock` script reads and writes a local JSON file. Both run with whatever permissions the person running them has.

The skills tell coding agents how to work. A malicious edit to a skill could steer an agent badly, so changes to `skills/` go through review and the CLA like anything else. Read a skill before installing it, same as you'd read a shell script.

Useful reports name the file, the concrete impact, and a short reproduction.
