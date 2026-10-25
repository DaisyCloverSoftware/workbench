#!/usr/bin/env bash
# Read-only diagnostic: two fixed GitHub.com runner-list GET endpoints.
# Usage: github-runner-routing.sh OWNER/REPO
# Run via Workbench's exact-commit operations channel; never prints credentials.
# Partial/denied inventories are UNKNOWN, not proof that no runner exists.
set -euo pipefail
exec python3 - "$@" <<'PY'
import datetime
import json
import os
import re
import shutil
import subprocess
import sys


def emit(value, code):
    print(json.dumps(value, indent=2))
    raise SystemExit(code)


if len(sys.argv) != 2 or not re.fullmatch(
    r"[A-Za-z0-9][A-Za-z0-9-]{0,38}/[A-Za-z0-9_][A-Za-z0-9_.-]{0,99}", sys.argv[1]
):
    emit({"status": "invalid_arguments", "usage": "github-runner-routing.sh OWNER/REPO"}, 2)

repository = sys.argv[1]
owner = repository.split("/", 1)[0]
gh = shutil.which("gh")
if not gh:
    emit({"status": "unavailable", "error_code": "github_cli_not_installed"}, 1)

environment = dict(os.environ)
environment.pop("GH_DEBUG", None)
environment["GH_PROMPT_DISABLED"] = "1"
projection = "{total_count: .total_count, runners: [.runners[] | {id, os, status, busy, labels: [.labels[].name]}]}"


def inspect(endpoint):
    command = [gh, "api", "--hostname", "github.com", "--method", "GET",
               endpoint + "?per_page=100&page=1", "--jq", projection]
    try:
        result = subprocess.run(command, capture_output=True, text=True,
                                encoding="utf-8", errors="replace", timeout=25,
                                env=environment, check=False)
    except (OSError, subprocess.TimeoutExpired):
        return {"status": "unavailable", "error_code": "github_request_unavailable"}
    if result.returncode:
        status = re.search(r"HTTP ([1-5][0-9]{2})", result.stderr)
        return {"status": "unavailable", "error_code":
                "github_http_" + status.group(1) if status else "github_request_failed"}
    try:
        if len(result.stdout) > 2 * 1024 * 1024:
            raise ValueError()
        payload = json.loads(result.stdout)
        total = payload["total_count"]
        runners = payload["runners"]
        if type(total) is not int or total < 0 or not isinstance(runners, list) or len(runners) > 100:
            raise ValueError()
        ids, safe = set(), []
        for runner in runners:
            rid = runner["id"]
            labels = runner["labels"]
            if type(rid) is not int or rid <= 0 or rid in ids or type(runner["busy"]) is not bool:
                raise ValueError()
            if not isinstance(labels, list) or not all(isinstance(label, str) for label in labels):
                raise ValueError()
            if not isinstance(runner["os"], str) or runner["status"] not in ("online", "offline"):
                raise ValueError()
            ids.add(rid)
            safe.append({key: runner[key] for key in ("id", "os", "status", "busy", "labels")})
        if len(safe) > total:
            raise ValueError()
    except (ValueError, TypeError, KeyError):
        return {"status": "unavailable", "error_code": "invalid_runner_response"}
    matching = [runner for runner in safe if runner["os"].lower() == "windows" and
                {"self-hosted", "windows", "x64"}.issubset({label.lower() for label in runner["labels"]})]
    return {"status": "complete" if len(safe) == total else "partial",
            "total_count": total, "returned_count": len(safe), "runners": safe,
            "matching_windows_x64_ids": [runner["id"] for runner in matching],
            "online_idle_matching_ids": [runner["id"] for runner in matching
                                         if runner["status"] == "online" and not runner["busy"]]}


scopes = {"repository": inspect("repos/" + repository + "/actions/runners"),
          "organization": inspect("orgs/" + owner + "/actions/runners")}
complete = all(scope["status"] == "complete" for scope in scopes.values())
matching = any(scope.get("matching_windows_x64_ids") for scope in scopes.values())
conclusion = ("matching_runner_visible_eligibility_not_proven" if matching else
              "no_matching_registered_runner_visible" if complete else "incomplete_inventory")
emit({"schema_version": 1, "repository": repository,
      "observed_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(),
      "read_only": True, "scopes": scopes, "routing_conclusion": conclusion,
      "note": "One page per scope, maximum 100 entries. Partial or denied reads cannot prove absence. Organization visibility does not prove repository/group eligibility. This does not establish historical runner loss or execute a build.",
      "native_unreal_proof": "not_obtained"}, 0 if complete else 1)
PY
