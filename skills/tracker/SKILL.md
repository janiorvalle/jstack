---
name: tracker
description: "Use to claim a task, record the files you'll touch, file a ticket, turn work in with the PR and evidence, or find out which tracker a repo uses. One contract everywhere, and a short section per backend the repo's Tracker line can name: markdown tasks in the repo, GitHub Issues, Linear, or Jira. Every ticket is four labels under 120 words, checked by the lint before it's filed."
---

# Tracker

Work lives in the tracker. The verbs are the same in every repo: claim, record the files, turn in, a human completes, evidence on the ticket. Only the backend differs, and the repo names it on one line. The sections at the bottom give each backend's commands.

## The line that names the tracker

A repo names its tracker on one line of its own `AGENTS.md`, or `CLAUDE.md` when that's the file it keeps. The line is `Tracker:`, the backend, then the one thing that backend needs:

```
Tracker: markdown tasks/
Tracker: github-issues
Tracker: linear SR
Tracker: jira SR
```

The folder for markdown. Nothing for GitHub Issues, since gh reads the repo from git. The team key for Linear. The project key for Jira. Find it with `grep -h '^Tracker:' AGENTS.md CLAUDE.md`.

No line means nobody has chosen yet. Ask the human in the Decide shape, one option per backend below, each asking for the one thing it needs. Never pick a default silently. Check first for an open PR from a branch named `tracker-line`, `gh pr list --head tracker-line --state open`. If one already carries a line, show it and ask the human to confirm that one instead.

Write the answer into the file, or a new `AGENTS.md` when the repo has neither, on its own line near the top, right under the title. Do it in its own worktree on the branch `tracker-line`, then offer to open the one-line PR for it with gh. A push refused because the branch appeared meanwhile means another agent asked first, so confirm its PR instead. The line is the one project file a claim needs, so writing it comes before the claim, and its own branch keeps it out of the task's diff. When the file is also a letter setup installs into harnesses, as jstack's own `AGENTS.md` is, setup leaves the `Tracker:` line out of the installed block, so the line stays with the repo.

## The five verbs

1. **Claim** before touching project files. A ticket with an owner is taken, so check first and pick another if it is. Put your name on it and read it back. Two agents can claim in the same second. If another name landed with yours or instead of yours, the earlier claim comment wins, and the other agent drops the claim and picks another. Open your first message with the ticket id.
2. **Record the files** you expect to change, right after claiming, in the claim comment or the frontmatter, never in the ticket body. A reviewer reads them against the diff.
3. **Turn in** with the PR link and the evidence. Then stop.
4. **A human completes** after the merge. Never the agent that built it.
5. **Evidence lives on the ticket.** Never in git.

## The ticket shape

Anyone gets it in twenty seconds. Four labels, at most 120 words:

```
Problem: Two sentences, from the person who hits it.
Fix: Two or three sentences.
Done when:
- Up to four observable lines.
Out of scope: Optional, one line.
```

Design notes go in the PR. The files you expect to touch go in the claim comment. Neither goes in the ticket body. The same shape is the PR description and the turn-in comment.

Run the lint before filing. It ships in this skill's `scripts/` folder, so give its full path from wherever your harness installed the skill. It reads a file or stdin and exits non-zero over 120 words, when a label is missing or empty, or when Done when isn't one to four lines starting with `- `. It says what to cut:

```sh
python3 <skill folder>/scripts/ticket-lint.py ticket.md
```

On Windows, where `python3` isn't a command, use `py -3`.

One that passes:

```
Problem: Opening a new thread ignores my "new worktree" default when I'm already in a worktree. I set the preference and nothing changes.
Fix: New threads inherit only the project from context. Branch, worktree, and env mode come from the configured defaults every time.
Done when:
- A new thread from inside a worktree lands in a fresh worktree.
- The preference screen's value is the one used.
Out of scope: The sidebar's thread list.
```

## Evidence

What counts as evidence is in `prove-it`. Where it goes is here: on the ticket, as comments with links and attachments, in whatever form the backend has. GitHub comments take images, video, PDFs, and zips but not HTML, so on GitHub the walkthrough goes one of two ways. In a public repo, a gist, `gh gist create walkthrough.html`, linked from the comment. In a private repo, a zip dropped on the comment, never a gist. A gist is readable by anyone with the URL, and the comment is covered by the repo's access rules. Nothing binary ever enters git, and the tasks folder is git too.

## Markdown tasks in the repo

`Tracker: markdown tasks/`. The word after `markdown` is the folder, `tasks/` here. One file per task, named for the task, under `tasks/open/`, `tasks/doing/`, and `tasks/done/`. The folder is the status. The frontmatter carries it too, with the owner, the files, and the PR:

```
---
status: doing
owner: agent:session-abc
files: [src/thread.ts, src/thread.test.ts]
pr: https://github.com/owner/repo/pull/12
---
Problem: ...
```

File: write it under `tasks/open/`, lint it, and land it on the default branch through a small PR with only that file, on a branch named `file-<task>`. When you're about to do the task yourself, file and claim in one go instead.

Claim: `git fetch` first and look for a remote branch named `<task>`, the task file's name. One already there means the task is taken, so pick another. A claim here is a commit, so the branch and worktree from `worktree` come before it, the one place the mode's order flips. Creating them touches no project file. Then move the file to `tasks/doing/` with your owner and files in the frontmatter. That's the first commit on the branch, pushed before any project file changes, so the branch is what the next agent finds.

Turn in: the PR link in the frontmatter and the file moved to `tasks/done/`, in the same PR as the work.

Complete: the merge, since the PR moves the file.

The PR is the turn-in comment. Screenshots and recordings go on as PR comment attachments, the walkthrough as a gist or a zip the way Evidence says.

## GitHub Issues

`Tracker: github-issues`. gh does all of it, and the issue number is the ticket id.

- File: `gh issue create --title "<outcome>" --body-file ticket.md`, after the lint.
- Claim: `gh issue view <n> --json assignees` first; an assignee there means it's taken. Then `gh issue edit <n> --add-assignee @me`, `gh issue comment <n> --body "Claimed. Files: src/thread.ts, src/thread.test.ts"`, and the same view again. GitHub allows several assignees, so a second name means the earlier claim comment wins and the other one runs `gh issue edit <n> --remove-assignee @me`.
- Turn in: `gh issue comment <n> --body-file turnin.md`, the ticket shape plus the PR link and the evidence links. Screenshots and recordings drop into that comment through the browser, since gh can't upload attachments. The walkthrough goes in a gist or a zip the way Evidence says.
- Complete: `gh issue close <n>` after the merge, by the human.

## Linear

`Tracker: linear SR`, where `SR` is the team key. Connect Linear's MCP in your harness; every verb is a call on it.

- File: create the issue in that team with the ticket shape as the description, after the lint.
- Claim: read the issue first; an assignee there means it's taken. Then post the claim comment with the files, assign yourself, move it to In Progress, and read the comments back. The earliest claim comment wins, since comments can't be reordered. If it isn't yours, drop the assignment and pick another.
- Turn in: a comment with the PR link and the evidence, screenshots, recordings, and the walkthrough attached, then the status to In Review, or whatever the team calls the state between building and merging.
- Complete: Done after the merge, by the human.

The PR title carries the issue key, `fix(web): SR-123 new threads respect the worktree default`, so Linear links the PR to the issue.

## Jira

`Tracker: jira SR`, where `SR` is the project key. Connect Jira's MCP in your harness; every verb is a call on it.

- File: create the issue in that project with the ticket shape as the description, after the lint.
- Claim: read the issue first; an assignee there means it's taken. Then post the claim comment with the files, assign yourself, transition it to In Progress, and read the comments back. The earliest claim comment wins, since comments can't be reordered. If it isn't yours, drop the assignment and pick another.
- Turn in: a comment with the PR link and the evidence, screenshots, recordings, and the walkthrough attached, then the transition to In Review, or whatever the project's workflow calls the state between building and merging. List the transitions the issue offers and pick by name, since the project may have renamed it.
- Complete: Done after the merge, by the human.

The PR title carries the issue key, `fix(web): SR-123 new threads respect the worktree default`, so Jira links the PR to the issue.

## Never

- Touch project files before the claim.
- File a ticket the lint rejects.
- Put a file list or design notes in the ticket body. Files go in the claim comment, design notes in the PR.
- Complete a ticket you built.
- Commit a screenshot, a recording, or a walkthrough.
