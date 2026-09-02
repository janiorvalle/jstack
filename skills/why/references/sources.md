# Sources

One section per evidence category. Give each investigator the section for its category, adapted to the tool actually available. The examples name common tools. Swap in whatever the team uses. The incident angle at the end gets added to every investigator when the target looks defensive.

Every section has the same shape. What the source holds, how to search it, what good evidence looks like, what goes wrong, what to return.

---

## 1. Source control

Git plus the PR host (GitHub, GitLab). Always available. The most trustworthy source because it's tied to the diff that shipped.

**What it holds.** Commit messages, dates, authors, diffs. PR descriptions, review comments, discussion threads. Inline comments, TODOs, FIXMEs, deprecation notes. ADRs if the repo keeps them. Tests, whose names often encode the edge case that motivated a change. Files changed together in the same commits. Changelogs. Ticket ids in commit messages and PR bodies.

**How to search.**

```bash
git log --follow --oneline -- <file>          # history through renames
git log -S '<exact string>' -- <file>          # commits that added or removed this text
git log -G '<regex>' -- <file>                 # same, by pattern
git blame -L <start>,<end> <file>              # who wrote each line, when
git show <hash>                                # full diff of one commit
git log -1 --format=%B <hash>                  # full message, PR number usually here
gh pr view <n> --json title,body,author,createdAt,mergedAt,labels,closingIssuesReferences,comments,reviews,files
rg -n -C2 '(TODO|FIXME|HACK|XXX|NOTE)' <file>  # comments near the target
rg -l '<symbol>' --glob '*test*'               # tests that name the symbol
rg -l -i 'architecture.decision' --glob '*.md' # ADRs
```

The reviews and comments fields on the PR are where the signal is.

**Good evidence.** A PR description that explains the problem, not just the change. A long review thread where alternatives were debated. An inline comment explaining a non-obvious constraint. A test named for the edge case. A commit message citing a ticket or incident.

**What goes wrong.** Squash merges lose the branch commits, fall back to the PR body and comments. "Small refactor" sometimes hides a behavior change, read the diff not the message. A pattern may have been copied without understanding, find where it first appeared and investigate that commit. Bot commits (dependabot, renovate, backports) carry no motivation, skip them. The code itself is not evidence of intent.

**Return.** Every commit, PR, or comment that bears on the question. Exact text quoted, hash or PR number or file and line, author and date, whether it's direct or circumstantial.

---

## 2. Issue tracker

Jira, Linear, GitHub Issues, Monday, Shortcut. Where the product and business reason usually lives. "Customer X asked for it." "Needed for the Q3 compliance push."

**What it holds.** Issues with descriptions and comments. Parent and child relationships, the parent often has the why. Attached specs. Labels that classify motivation (customer request, incident follow-up, compliance, perf). Status changes that explain scope shifts. Links to PRs.

**How to search.**

1. Start with ids from the anchor. Fetch each in full, including comments.
2. Keyword search for the feature name, key symbols, and business terms. Try several phrasings.
3. Walk up. If you land on a subtask, fetch its parent. Subtasks are tactical, parents carry the reason.
4. Read project or epic docs if the issue belongs to one.
5. Check labels and milestones. A deadline often reveals the motivation.

**Good evidence.** A description stating the business problem. A comment recording a decision: "went with B because A would touch billing". A parent issue named like an initiative. An attached spec. A label like customer:acme or incident-followup.

**What goes wrong.** Scope drift, the ticket was closed and reopened with a different scope, read the whole history. Template boilerplate, "improve user experience" is not a reason. Stale tickets describing a plan that changed, check dates against the ship date. Duplicate chains, follow duplicate-of back to the canonical one. No access, note it as a gap.

**Return.** For each relevant issue: id and title, the motivation quoted from the description or a comment, labels and parent, author and dates, link.

---

## 3. Long-form docs

Confluence, Notion, Google Docs. Where the reason is written out before it becomes code, for anything big enough to get a doc.

**What it holds.** PRDs, specs, RFCs, ADRs, design review notes, team pages, postmortems, runbooks, strategy docs.

**How to search.**

1. Keyword search for the feature name, key symbols, author names, error strings, user-visible terms. Time-bound if you know the ship date.
2. Fetch the full page. The reason is usually mid-document, not in the preview.
3. Follow child pages and backlinks. Alternatives considered often lives in a sub-page.
4. Check meeting notes databases if the tool has them.
5. Check the author's personal space if one exists. Exploratory thinking often lives there.

**Good evidence.** A problem statement or motivation section matching the target. An alternatives-considered or rejected-approaches section. A postmortem naming the target as the fix for an incident. Meeting notes recording "decided X because Y" in the same date range as the PR. An ADR filled in non-trivially.

**What goes wrong.** The doc describes the plan, the code does something else, flag the divergence. Boilerplate why sections. The relevant doc isn't linked from anywhere, search broadly. Several drafts, find the final one by date. No access, gap.

**Return.** For each doc: title and URL, authors and last updated, the motivation quoted with section, linked pages, draft or final.

---

## 4. Team chat

Slack, Teams, Discord. Where the real decisions often happen, especially for changes too small for a doc. Also the most fragile source. Threads get deleted, channels archived, search degrades.

**What it holds.** Real-time discussion of problems and decisions. Incident channels. Design threads. Questions answered by seniors that never reached a doc. Post-merge discussion of why something was revisited.

**How to search.**

1. Messages from the PR author around the merge date. Narrow and often gold.
2. Keyword search for the feature name and symbols, including casual spellings.
3. Search for the PR URL or `/pull/<n>`. PRs get linked when discussed.
4. Search for error strings the code handles. Incident threads surface.
5. Scope to likely channels: engineering, project, incident and sev channels, the owning team's channel.
6. When you find a message, fetch the whole thread. The decision is in the replies.

**Good evidence.** A thread where tradeoffs were debated. An incident channel message describing the bug this prevents. A reviewer question with an authoritative answer. A PM or support engineer explaining a customer ask.

**What goes wrong.** Retention cliffs, note the date before which nothing exists. DMs aren't searchable, a known blind spot. "Lol just do it" isn't a decision. A single message without its thread reads differently. If the tool needs auth and fails, stop and report the gap. Don't invent.

**Return.** For each thread: channel, permalink, participants, date range, key quotes verbatim with attribution, what the thread was about.

---

## 5. Observability

Datadog, Grafana, New Relic, Honeycomb. The runtime record. What actually happened in production, versus what was planned.

**What it holds.** Metrics, and a metric's existence is itself evidence someone cared. Monitors and alerts, the thresholds a team decided were worth waking someone for. Dashboards. Traces and spans. Logs. Formal incidents with timelines. Notebooks with investigations.

**How to search.**

1. Find the owning service and its dependencies.
2. Dashboards and monitors first. They say what the team watches. A monitor threshold that matches a code constant is often the answer to "why is this clamped at N".
3. Metrics near the target. Was there a spike right before the PR merged, stable after?
4. Logs, narrowed hard. Symbols, error strings, feature names, always time-bounded to a window around the change. Aggregate rather than dump.
5. Spans and traces for timeouts, retries, slow paths, cross-service behavior.
6. Incidents around the date the target was added, especially if it looks defensive.

**Good evidence.** A monitor whose query and threshold match what the code enforces. A dashboard by the target's author with widgets matching what the code measures. A metric spiking right before the merge and flat after. An incident referencing the target or its error strings. Logs showing the exact error pattern in the window before the change.

**What goes wrong.** Correlation isn't causation, other changes landed in the same window, check neighbors. A chart reflects its author's framing, not the reason for a line of code. Metrics get renamed or expire, missing data is a gap not a null. Common strings return thousands of logs, narrow by service, tag, time. Instrumented doesn't mean caused, cross-reference with commit dates.

**Return.** For each item: type, name, id or link, owner and dates, the specific condition or query or quote, how strongly it connects to the target.

---

## 6. Error tracking

Sentry, Rollbar, Bugsnag, PostHog error tracking. The archive of what went wrong. For defensive or corrective code, often holds the direct motivation: the exception, the stack trace, the frequency that made someone add the check.

**What it holds.** Issues grouped by fingerprint with counts and first and last seen. Individual events with stack traces and context. Releases with associated issues. Session replays. Issue comments and assignments, sometimes with root-cause notes.

The most useful thing here is timing. "Issue X first seen Jan 2, peaked at 500 a day, stopped after the release on Jan 15 that shipped the check."

**How to search.**

1. Search issues by the exception class the target handles, the function or class name, error message strings, the file path.
2. For each candidate, check first seen, last seen, affected releases, and the count trajectory against the target's ship date.
3. Pull full events. Does the stack trace pass through the target? Do tags and breadcrumbs match what it defends against?
4. Find releases near the target's commit date and cross-reference.
5. If the tool offers AI root-cause analysis, treat it as a hypothesis, not evidence. The events and traces are primary.

**Good evidence.** An issue first seen shortly before the PR and last seen shortly after. A stack trace through the target function. An author comment on the issue describing the fix. The PR referencing the issue URL. High counts that stop after the release containing the target.

**What goes wrong.** Refactors regroup the same error under a new id, an issue ending abruptly may have just moved. A release has many commits, stopping at v2.14 doesn't prove this commit did it. The error may have stopped because upstream changed. "Resolved" is a human marker, not proof of a fix. Sampling makes low counts misleading.

**Return.** For each issue: id and title, project, first and last seen, count and sampling if known, releases, a stack trace excerpt showing relevance, timing against the ship date, link, any author notes.

---

## 7. Product analytics

PostHog, a warehouse like Databricks, Snowflake, or BigQuery, experiment and flag tables. The product and data view, next to observability's infra view. What users did, which experiments ran, how usage evolved, where a threshold number came from.

**What it holds.** Product events, feature usage, clicks, submissions, client-reported errors. Usage and billing events. Experiment exposures and flag data, schema varies by company. Query history and warehouse system tables. Pipeline lineage. Notebooks, usually not queryable, name them as a gap if you suspect the reason is in one.

**How to search.**

Orient first. Schemas differ. List tables and describe them before trusting a name. Time-bound every query, a window around the ship date, usually 30 days each side. Prefer typed models over raw event tables. Never dump rows, return counts and percentiles.

Patterns that pay off:

1. **Usage trajectory.** Daily counts of the relevant event across the window. A step from zero to steady volume within a day of the merge says the PR launched it. A decay to zero says deprecation.
2. **Where a threshold came from.** The distribution (median, p99, max) of the relevant property in the two weeks before the PR. A p99 matching the code's constant says the number came from data.
3. **Experiment or flag.** Find the exposure table, pull counts by variant for the flag key near the PR date.
4. **Migrations and perf rewrites.** Query history filtered by the table or symbol in a tight window surfaces the expensive queries that motivated the change.
5. **Lineage.** If the target reads or writes a warehouse model, that model's git history often has the reason. Hand that lead to the source control investigator.

**Good evidence.** An error-classifying event dropping to near zero after a defensive PR. An exposure row naming the target's flag with a shipped decision near the date. A pre-ship distribution whose p99 equals the constant in the code.

**What goes wrong.** Instrumented doesn't mean caused, pair with a commit citation. A step in volume may mean logging started, not behavior changed, check for instrumentation PRs in the window. Schema drift, the property may not have existed then. Refresh lag on modeled tables. Reporting from a table you never confirmed exists. The window predates retention, that's a gap not a null.

**Return.** For each finding: type, fully-qualified table and the exact query, time window, compact numbers, timing against the ship date, direct or circumstantial or weak.

---

## The incident angle

Not a separate source. An extra lens every investigator applies when the target looks defensive: null checks, retries, timeouts, rate limits, feature flags, size clamps, memory guards. Defensive code is often the action item from an outage.

Inside your own source, look for incident history:

- Source control: commits saying "fix for incident", "add defensive check", a revert followed by a re-apply with changes.
- Issue tracker: labels like incident, sev, postmortem action item, reliability.
- Docs: postmortems mentioning the file, feature, or error string.
- Chat: incident and sev channels around the dates the target was added.
- Observability: formal incidents with timelines, dashboards and monitors created as postmortem follow-ups.
- Error tracking: issues whose first and last seen bracket the ship date, traces through the target.
- Product analytics: an error-classified event spiking during an incident window and dropping after the target shipped.

If you find an incident link, fetch the full postmortem. The action items section ties directly to code changes. When several sources corroborate, an incident id in a ticket, in a postmortem, in a chat thread linking the PR, and an event count dropping after the fix, the evidence is strong.

Skip this angle for code that doesn't look defensive.
