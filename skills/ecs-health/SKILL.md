---
name: ecs-health
description: "Use for \"check ECS for rollbacks\", \"any Fargate services in a bad state\", \"stuck deployments\", \"desired and running don't match\", \"check target health\", across one AWS profile, a group of them, or all of them. Read-only audit of ECS Fargate services for rollback events and current bad health. Never changes anything."
---

# ECS health

A read-only sweep of ECS Fargate services across your AWS profiles. It reports two things: services in a bad state right now, and rollback events ECS still has in the service history. Nothing else. It never touches a service, a deployment, a target group, or any AWS resource.

## Where the profiles come from

`~/.config/jstack/ecs-health.json`. `setup-jstack` creates it from every profile in `~/.aws/config` and adds new profiles on later runs without touching your tags. You tag profiles once:

```json
{
  "profiles": [
    { "name": "acme", "group": "customer", "label": "Acme Corp" },
    { "name": "acme-prod", "group": "customer" },
    { "name": "personal", "group": "personal" },
    { "name": "logging", "skip": true }
  ],
  "environments": {
    "prod": ["prod", "production"],
    "dev": ["dev", "test", "staging"]
  }
}
```

`skip` keeps a profile out of every scan unless it's named with `--profiles`. `group` lets you scan a slice with `--group customer`. `environments` is the only place that says which service prefixes count as prod and which as dev. There are no built-in lists. `--scope prod` and `--scope dev` refuse to run until it's filled in. `--scope nonprod` and `--scope all` work without it. With no config file at all, every profile in `~/.aws/config` gets scanned, ungrouped.

An environment is the part of a service name before its first hyphen. `staging-api` is in `staging`.

## Scope

Work out the scope from the request before running. If it's stated, don't ask:

- One profile: `--profiles acme`
- One profile and environment: `--profiles acme --environments staging`
- All non-production: `--scope nonprod` (the default)
- All production: `--scope prod`
- Everything: `--scope all`
- Dev-like only: `--scope dev`
- Several groups: `--scope dev prod`
- One tag from the config: `--group customer`

If the scope isn't stated, ask one question: a specific profile or environment, everything, or a group like dev, nonprod, prod, or all. Production is included only when the request says prod, production, or all. It's still read-only either way.

## Run it

```bash
python3 scripts/ecs_health.py --scope nonprod
python3 scripts/ecs_health.py --profiles acme --environments staging --service-regex '^staging-api$'
python3 scripts/ecs_health.py --group customer --scope prod --json /tmp/ecs-health.json
```

Other flags. `--region` (default `us-east-1`). `--include-skipped`. `--skip-target-health` when ALB checks aren't needed. `--max-workers` for parallel profiles. `--fail-on-findings` exits non-zero when anything is bad, for scripts.

If SSO is expired, run `aws sso login --profile <name>` for the requested profiles only, then retry.

## What counts as bad

Current state, any of:

- service not `ACTIVE`
- desired and running counts don't match
- pending tasks
- primary deployment not `COMPLETED`
- failed tasks on the primary deployment
- a retained failed deployment
- unhealthy or missing ALB target health when the service has a target group

Rollback events: anything in the retained ECS event history matching `rollback`, `roll back`, `rolled back`, or `rolling back`. The last one matters. ECS writes `(service staging-api) rolling back to deployment ecs-svc/...`.

Fargate and Fargate Spot services only. Everything else is skipped.

## Reporting

Lead with what's bad now and any rollback events. For each, give profile, cluster, service, compute type, desired, running, pending, rollout state, task definition, timestamp, and reason. Leave unrelated deployment history out unless asked for raw events. If a service rolled back but is healthy now, say both. If target health changes during the scan, recheck that one service before reporting.
