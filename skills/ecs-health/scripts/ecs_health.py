#!/usr/bin/env python3
"""Read-only ECS Fargate health and rollback audit across AWS profiles.

Profiles come from ~/.config/jstack/ecs-health.json (setup-jstack creates and
updates it from ~/.aws/config). With no config file, every profile in
~/.aws/config is scanned.
"""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path
from typing import Any


DEFAULT_REGION = "us-east-1"
CONFIG_PATH = Path.home() / ".config" / "jstack" / "ecs-health.json"
AWS_CONFIG_PATH = Path.home() / ".aws" / "config"
PRODUCTION_RE = re.compile(r"(^|[-_/])prod(uction)?($|[-_/])|production", re.I)
ROLLBACK_RE = re.compile(r"rollback|roll back|rolled back|rolling back", re.I)
PROD_ENVS = {"prod", "production"}
DEV_ENVS = {"dev", "test", "demo", "staging", "uiux", "trial", "s2s", "batest", "customer", "anon"}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Read-only ECS Fargate rollback and health audit across AWS profiles."
    )
    parser.add_argument("--config", default=str(CONFIG_PATH), help="Profile config file. Falls back to every profile in ~/.aws/config.")
    parser.add_argument("--group", nargs="*", help="Only scan profiles tagged with these groups in the config.")
    parser.add_argument("--region", default=DEFAULT_REGION, help="AWS region to scan.")
    parser.add_argument("--profiles", nargs="*", help="Specific AWS CLI profiles to scan.")
    parser.add_argument(
        "--scope",
        nargs="+",
        default=["nonprod"],
        choices=["nonprod", "prod", "dev", "all"],
        help="Environment group(s) to scan. Defaults to nonprod.",
    )
    parser.add_argument(
        "--environments",
        nargs="*",
        help="Specific service environment prefixes to scan, such as training, config, support, or production.",
    )
    parser.add_argument("--include-skipped", action="store_true", help="Include profiles marked skip in the config.")
    parser.add_argument("--service-regex", help="Only scan services whose name matches this regex.")
    parser.add_argument("--skip-target-health", action="store_true", help="Skip ALB target health checks.")
    parser.add_argument("--max-workers", type=int, default=6, help="Parallel account scans.")
    parser.add_argument("--json", dest="json_path", help="Write full result JSON to this path.")
    parser.add_argument(
        "--fail-on-findings",
        action="store_true",
        help="Exit non-zero when current bad states or rollback events are found.",
    )
    return parser.parse_args()


def run_aws(profile: str, region: str, args: list[str], timeout: int = 90) -> tuple[dict[str, Any] | None, str | None]:
    cmd = ["aws", *args, "--profile", profile, "--region", region, "--output", "json"]
    proc = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout)
    if proc.returncode != 0:
        return None, (proc.stderr or proc.stdout).strip()
    try:
        return json.loads(proc.stdout or "{}"), None
    except json.JSONDecodeError as exc:
        return None, f"Could not parse AWS JSON output: {exc}"


def arn_tail(value: Any) -> str:
    return str(value or "").split("/")[-1]


def is_production_name(name: str) -> bool:
    return bool(PRODUCTION_RE.search(name or ""))


def environment_name(service_name: str) -> str:
    return (service_name.split("-", 1)[0] or "").lower()


def normalized_environment_filters(environments: list[str] | None) -> set[str]:
    normalized: set[str] = set()
    for environment in environments or []:
        value = environment.strip().lower()
        if not value:
            continue
        normalized.add(value)
        if value == "prod":
            normalized.add("production")
        if value == "production":
            normalized.add("prod")
    return normalized


def service_matches_scope(cluster_name: str, service_name: str, args: argparse.Namespace) -> bool:
    prod_envs = set(getattr(args, "prod_envs", PROD_ENVS))
    dev_envs = set(getattr(args, "dev_envs", DEV_ENVS))
    env = environment_name(service_name)
    explicit_envs = normalized_environment_filters(args.environments)
    if explicit_envs:
        return env in explicit_envs

    scopes = set(args.scope or ["nonprod"])
    if "all" in scopes:
        return True

    prodish = env in prod_envs or is_production_name(cluster_name) or is_production_name(service_name)
    if "prod" in scopes and prodish:
        return True
    if "nonprod" in scopes and not prodish:
        return True
    if "dev" in scopes and not prodish and env in dev_envs:
        return True
    return False


def is_fargate_service(service: dict[str, Any]) -> bool:
    if service.get("launchType") == "FARGATE":
        return True
    return any(
        "FARGATE" in (item.get("capacityProvider") or "")
        for item in service.get("capacityProviderStrategy") or []
    )


def compute_label(service: dict[str, Any]) -> str:
    providers = service.get("capacityProviderStrategy") or []
    if providers:
        return ",".join(item.get("capacityProvider", "?") for item in providers)
    return service.get("launchType") or "unknown"


def chunks(items: list[str], size: int) -> list[list[str]]:
    return [items[index : index + size] for index in range(0, len(items), size)]


def aws_config_profiles() -> list[str]:
    """Every [profile X] and the [default] section in ~/.aws/config."""
    if not AWS_CONFIG_PATH.is_file():
        return []
    names: list[str] = []
    for line in AWS_CONFIG_PATH.read_text().splitlines():
        match = re.match(r"^\s*\[\s*(?:profile\s+)?([^\]]+?)\s*\]\s*$", line)
        if match:
            names.append(match.group(1))
    return sorted(set(names))


def load_config(path: str) -> dict[str, Any]:
    config_file = Path(path)
    if not config_file.is_file():
        return {"profiles": [{"name": name, "group": ""} for name in aws_config_profiles()]}
    return json.loads(config_file.read_text())


def load_accounts(args: argparse.Namespace) -> list[dict[str, str]]:
    config = load_config(args.config)
    args.prod_envs = set(config.get("environments", {}).get("prod", sorted(PROD_ENVS)))
    args.dev_envs = set(config.get("environments", {}).get("dev", sorted(DEV_ENVS)))
    requested = set(args.profiles or [])
    groups = set(args.group or [])
    accounts: list[dict[str, str]] = []
    for entry in config.get("profiles", []):
        profile = entry.get("name")
        if not profile:
            continue
        if requested and profile not in requested:
            continue
        if not requested and not args.include_skipped and entry.get("skip"):
            continue
        if groups and entry.get("group", "") not in groups:
            continue
        accounts.append(
            {
                "profile": profile,
                "account": str(entry.get("account") or ""),
                "alias": str(entry.get("group") or ""),
                "pretty": str(entry.get("label") or profile),
            }
        )
    accounts.sort(key=lambda item: item["profile"])
    return accounts


def primary_deployment(service: dict[str, Any]) -> dict[str, Any]:
    deployments = service.get("deployments") or []
    for deployment in deployments:
        if deployment.get("status") == "PRIMARY":
            return deployment
    return deployments[0] if deployments else {}


def target_health(profile: str, region: str, service: dict[str, Any]) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    for lb in service.get("loadBalancers") or []:
        target_group_arn = lb.get("targetGroupArn")
        if not target_group_arn:
            continue
        data, error = run_aws(
            profile,
            region,
            ["elbv2", "describe-target-health", "--target-group-arn", target_group_arn],
            timeout=60,
        )
        if error:
            rows.append({"targetGroup": arn_tail(target_group_arn), "error": error})
            continue
        for description in data.get("TargetHealthDescriptions", []):
            target = description.get("Target", {})
            health = description.get("TargetHealth", {})
            rows.append(
                {
                    "targetGroup": arn_tail(target_group_arn),
                    "target": target.get("Id"),
                    "port": target.get("Port"),
                    "state": health.get("State"),
                    "reason": health.get("Reason"),
                    "description": health.get("Description"),
                }
            )
    return rows


def scan_account(account: dict[str, str], args: argparse.Namespace, service_re: re.Pattern[str] | None) -> dict[str, Any]:
    profile = account["profile"]
    result: dict[str, Any] = {
        **account,
        "checked": 0,
        "clusters": 0,
        "ecs_services_seen": 0,
        "current_bad": [],
        "rollback_events": [],
        "errors": [],
    }
    clusters, error = run_aws(profile, args.region, ["ecs", "list-clusters"])
    if error:
        result["errors"].append(f"list-clusters: {error}")
        return result
    cluster_arns = clusters.get("clusterArns", [])
    result["clusters"] = len(cluster_arns)

    for cluster_arn in cluster_arns:
        cluster_name = arn_tail(cluster_arn)
        services, error = run_aws(profile, args.region, ["ecs", "list-services", "--cluster", cluster_arn])
        if error:
            result["errors"].append(f"{cluster_name}: list-services: {error}")
            continue
        service_arns = services.get("serviceArns", [])
        result["ecs_services_seen"] += len(service_arns)
        for batch in chunks(service_arns, 10):
            described, error = run_aws(
                profile,
                args.region,
                ["ecs", "describe-services", "--cluster", cluster_arn, "--services", *batch],
            )
            if error:
                result["errors"].append(f"{cluster_name}: describe-services: {error}")
                continue
            for failure in described.get("failures") or []:
                result["errors"].append(f"{cluster_name}: describe failure: {failure}")
            for service in described.get("services") or []:
                service_name = service.get("serviceName") or arn_tail(service.get("serviceArn"))
                if not service_matches_scope(cluster_name, service_name, args):
                    continue
                if service_re and not service_re.search(service_name):
                    continue
                if not is_fargate_service(service):
                    continue
                inspect_service(result, profile, cluster_name, service_name, service, args)
    return result


def inspect_service(
    result: dict[str, Any],
    profile: str,
    cluster_name: str,
    service_name: str,
    service: dict[str, Any],
    args: argparse.Namespace,
) -> None:
    result["checked"] += 1
    primary = primary_deployment(service)
    desired = int(service.get("desiredCount") or 0)
    running = int(service.get("runningCount") or 0)
    pending = int(service.get("pendingCount") or 0)
    rollout = primary.get("rolloutState") or "UNKNOWN"
    failed_tasks = int(primary.get("failedTasks") or 0)

    base = {
        "profile": profile,
        "cluster": cluster_name,
        "service": service_name,
        "compute": compute_label(service),
        "desired": desired,
        "running": running,
        "pending": pending,
        "rollout": rollout,
        "taskDefinition": arn_tail(service.get("taskDefinition")),
    }

    reasons: list[str] = []
    if service.get("status") != "ACTIVE":
        reasons.append(f"service status {service.get('status')}")
    if desired != running:
        reasons.append(f"desired/running {desired}/{running}")
    if pending:
        reasons.append(f"pending {pending}")
    if rollout != "COMPLETED":
        reasons.append(f"rollout {rollout}")
    if failed_tasks:
        reasons.append(f"primary failedTasks {failed_tasks}")
    for deployment in service.get("deployments") or []:
        if deployment.get("rolloutState") == "FAILED":
            reasons.append(f"failed deployment retained {deployment.get('id')}")

    if not args.skip_target_health:
        health_rows = target_health(profile, args.region, service)
        has_target_group = bool(service.get("loadBalancers") or [])
        if has_target_group and desired > 0 and not health_rows:
            reasons.append("no targets registered")
        for row in health_rows:
            if row.get("error"):
                reasons.append(f"target health check error {row.get('targetGroup')}")
            elif desired > 0 and row.get("state") != "healthy":
                parts = [f"target {row.get('target')}:{row.get('port')}", str(row.get("state"))]
                if row.get("reason"):
                    parts.append(str(row["reason"]))
                reasons.append(" ".join(parts))

    if reasons:
        result["current_bad"].append({**base, "reasons": reasons})

    for event in service.get("events") or []:
        message = event.get("message") or ""
        if ROLLBACK_RE.search(message):
            result["rollback_events"].append(
                {
                    **base,
                    "createdAt": event.get("createdAt"),
                    "message": " ".join(message.split()),
                }
            )


def print_report(results: list[dict[str, Any]], elapsed: float, args: argparse.Namespace) -> None:
    current_bad = sorted(
        [item for result in results for item in result["current_bad"]],
        key=lambda item: (item["profile"], item["service"]),
    )
    rollback_events = sorted(
        [item for result in results for item in result["rollback_events"]],
        key=lambda item: item.get("createdAt") or "",
        reverse=True,
    )
    errors = [(result["profile"], error) for result in results for error in result["errors"]]

    print(
        "SUMMARY "
        + json.dumps(
            {
                "accounts": len(results),
                "clusters": sum(result["clusters"] for result in results),
                "ecsServicesSeen": sum(result["ecs_services_seen"] for result in results),
                "checkedFargate": sum(result["checked"] for result in results),
                "currentBad": len(current_bad),
                "rollbackEvents": len(rollback_events),
                "errors": len(errors),
                "durationSeconds": round(elapsed, 1),
                "scope": args.scope,
                "environments": args.environments or [],
            },
            sort_keys=True,
        )
    )

    print(f"CURRENT_BAD {len(current_bad)}")
    for item in current_bad:
        print(
            "\t".join(
                [
                    "CURRENT_BAD",
                    item["profile"],
                    item["cluster"],
                    item["service"],
                    item["compute"],
                    f"{item['desired']}/{item['running']}/{item['pending']}",
                    item["rollout"],
                    item["taskDefinition"],
                    "; ".join(item["reasons"]),
                ]
            )
        )

    print(f"ROLLBACK_EVENTS {len(rollback_events)}")
    for item in rollback_events:
        print(
            "\t".join(
                [
                    "ROLLBACK",
                    item["profile"],
                    item["cluster"],
                    item["service"],
                    item["compute"],
                    f"{item['desired']}/{item['running']}/{item['pending']}",
                    item["rollout"],
                    item["taskDefinition"],
                    str(item.get("createdAt") or ""),
                    item.get("message") or "",
                ]
            )
        )

    for profile, error in errors:
        print("\t".join(["ERROR", profile, error]))


def main() -> int:
    args = parse_args()
    service_re = re.compile(args.service_regex) if args.service_regex else None
    accounts = load_accounts(args)
    if not accounts:
        print("No accounts matched the requested filters.", file=sys.stderr)
        return 1

    started = time.time()
    results: list[dict[str, Any]] = []
    workers = max(1, args.max_workers)
    scope_label = ",".join(args.scope or [])
    environment_label = ",".join(args.environments or [])
    print(
        f"SCAN_START accounts={len(accounts)} region={args.region} scope={scope_label} environments={environment_label}",
        flush=True,
    )
    with ThreadPoolExecutor(max_workers=workers) as pool:
        future_map = {pool.submit(scan_account, account, args, service_re): account for account in accounts}
        for index, future in enumerate(as_completed(future_map), 1):
            result = future.result()
            results.append(result)
            print(
                "SCAN_PROGRESS "
                f"{index}/{len(accounts)} {result['profile']} "
                f"checked={result['checked']} currentBad={len(result['current_bad'])} "
                f"rollbacks={len(result['rollback_events'])} errors={len(result['errors'])}",
                flush=True,
            )
    results.sort(key=lambda item: item["profile"])
    elapsed = time.time() - started
    print_report(results, elapsed, args)

    if args.json_path:
        Path(args.json_path).write_text(json.dumps({"results": results}, indent=2, sort_keys=True) + "\n")

    findings = sum(len(item["current_bad"]) + len(item["rollback_events"]) for item in results)
    errors = sum(len(item["errors"]) for item in results)
    if errors:
        return 1
    if args.fail_on_findings and findings:
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
