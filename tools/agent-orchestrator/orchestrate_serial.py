#!/usr/bin/env python3
"""
Archived Serial Agent Orchestrator — historical inventory only.

The executable workflow was isolated on 2026-07-30. Only --dry-run remains
available for auditing the historical task list. All mutating modes fail closed
before reading Agent commands or changing Git state.

Workflow per task:
  1. Checkout integration/agent-platform
  2. Create task branch
  3. Run Developer Agent
  4. Verify commits exist
  5. Run Reviewer Agent → parse PASS / NEEDS_FIX / BLOCKED
  6. If NEEDS_FIX → Fixer Agent → re-Review (up to max_fix_rounds)
  7. If PASS → squash-merge to integration/agent-platform, delete task branch
  8. If BLOCKED or failure → stop (or continue if --continue-on-failure)
  9. Log everything to logs/

Usage:
  python orchestrate_serial.py --dry-run
  python orchestrate_serial.py --only P0-002
  python orchestrate_serial.py --from-task P0-002 --max-tasks 3
  python orchestrate_serial.py --no-merge --yes
"""

import argparse
import json
import os
import re
import subprocess
import sys
import time
from datetime import datetime, timezone
from pathlib import Path

# ---------------------------------------------------------------------------
# Paths
# ---------------------------------------------------------------------------
SCRIPT_DIR = Path(__file__).resolve().parent
REPO_ROOT = SCRIPT_DIR.parent.parent
TASKS_YAML = SCRIPT_DIR / "tasks.yaml"
LOGS_DIR = SCRIPT_DIR / "logs"
PROMPTS_DIR = SCRIPT_DIR / "prompts"

DEV_PROMPT_TEMPLATE = PROMPTS_DIR / "developer.md"
REVIEW_PROMPT_TEMPLATE = PROMPTS_DIR / "reviewer.md"
FIX_PROMPT_TEMPLATE = PROMPTS_DIR / "fixer.md"

STATUS_FILE = LOGS_DIR / "status.jsonl"
INTEGRATION_BRANCH = "integration/agent-platform"
ARCHIVED_MESSAGE = (
    "agent-orchestrator is archived and read-only; only --dry-run is allowed. "
    "Use AGENTS.md and an explicitly activated current repository skill instead."
)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def run(cmd: list[str], check: bool = True, capture: bool = True) -> subprocess.CompletedProcess:
    """Run a shell command and return the CompletedProcess."""
    kwargs: dict = {}
    if capture:
        kwargs["stdout"] = subprocess.PIPE
        kwargs["stderr"] = subprocess.PIPE
    result = subprocess.run(cmd, cwd=str(REPO_ROOT), text=True, **kwargs)
    if check and result.returncode != 0:
        msg = f"Command failed (exit {result.returncode}): {' '.join(cmd)}\nSTDERR:\n{result.stderr}"
        raise RuntimeError(msg)
    return result


def load_yaml_tasks(path: Path) -> list[dict]:
    """Load tasks from YAML file with minimal parsing (no PyYAML dependency)."""
    import yaml  # deferred import — fail early with a clear message if missing
    with open(path, "r", encoding="utf-8") as fh:
        data = yaml.safe_load(fh)
    if not data or "tasks" not in data:
        raise ValueError(f"tasks.yaml missing top-level 'tasks' key: {path}")
    return data["tasks"]


def check_branch(expected: str) -> bool:
    """Return True if HEAD is exactly `expected`."""
    out = run(["git", "rev-parse", "--abbrev-ref", "HEAD"]).stdout.strip()
    return out == expected


def check_clean_worktree() -> bool:
    """Return True if working tree is clean (no modified / untracked files)."""
    out = run(["git", "status", "--porcelain"]).stdout.strip()
    return out == ""


def has_commits_ahead(branch: str, base: str) -> bool:
    """Return True if `branch` has commits that `base` does not."""
    out = run(["git", "log", f"{base}..{branch}", "--oneline"]).stdout.strip()
    return bool(out)


def task_file_exists(task_file: str) -> bool:
    """Check whether the task markdown file exists under repo root."""
    return (REPO_ROOT / task_file).is_file()


def render_prompt(template_path: Path, variables: dict[str, str]) -> str:
    """Read a prompt template and replace {{VAR}} placeholders."""
    text = template_path.read_text(encoding="utf-8")
    for key, value in variables.items():
        text = text.replace("{{" + key + "}}", value)
    # Warn about any unreplaced placeholders
    remaining = re.findall(r"\{\{(\w+)\}\}", text)
    if remaining:
        print(f"  [WARN] Unreplaced placeholders in prompt: {remaining}")
    return text


def call_agent(prompt: str, agent_cmd: str) -> str:
    """Pipe `prompt` into `agent_cmd` and return stdout."""
    # agent_cmd is a shell command string, e.g. "claude -p"
    proc = subprocess.run(
        agent_cmd,
        shell=True,
        input=prompt,
        cwd=str(REPO_ROOT),
        text=True,
        capture_output=True,
    )
    output = proc.stdout
    if proc.returncode != 0:
        output += f"\n[AGENT EXIT CODE: {proc.returncode}]\n[STDERR]\n{proc.stderr}"
    return output


def parse_review_verdict(log_text: str) -> str | None:
    """Extract REVIEW_VERDICT from reviewer output. Returns PASS / NEEDS_FIX / BLOCKED / None."""
    match = re.search(r"REVIEW_VERDICT:\s*(PASS|NEEDS_FIX|BLOCKED)", log_text)
    if match:
        return match.group(1)
    return None


def write_status(entry: dict) -> None:
    """Append one JSON line to status.jsonl."""
    entry["timestamp"] = datetime.now(timezone.utc).isoformat()
    with open(STATUS_FILE, "a", encoding="utf-8") as fh:
        fh.write(json.dumps(entry, ensure_ascii=False) + "\n")


def log_file_path(task_id: str, stage: str, round_num: int | None = None) -> Path:
    """Build log file path. e.g. logs/P0-002-developer.log, logs/P0-002-review-2.log"""
    if round_num:
        return LOGS_DIR / f"{task_id}-{stage}-{round_num}.log"
    return LOGS_DIR / f"{task_id}-{stage}.log"


# ---------------------------------------------------------------------------
# Task execution steps
# ---------------------------------------------------------------------------

def step_checkout_integration() -> None:
    run(["git", "checkout", INTEGRATION_BRANCH])


def step_create_branch(branch_name: str) -> None:
    # Delete local branch if it already exists (stale from previous failed run)
    r = run(["git", "branch", "--list", branch_name], check=False)
    if r.stdout.strip():
        run(["git", "branch", "-D", branch_name])
    run(["git", "checkout", "-b", branch_name])


def step_delete_branch(branch_name: str) -> None:
    run(["git", "checkout", INTEGRATION_BRANCH])
    run(["git", "branch", "-D", branch_name])


def step_squash_merge(branch_name: str, commit_message: str) -> None:
    run(["git", "checkout", INTEGRATION_BRANCH])
    run(["git", "merge", "--squash", branch_name])
    run(["git", "commit", "-m", commit_message])


# ---------------------------------------------------------------------------
# Main orchestrator
# ---------------------------------------------------------------------------

def orchestrate(args: argparse.Namespace) -> int:
    if not args.dry_run:
        print(f"ERROR: {ARCHIVED_MESSAGE}")
        return 2

    # --- load tasks ---
    all_tasks = load_yaml_tasks(TASKS_YAML)
    enabled_tasks = [t for t in all_tasks if t.get("enabled", True)]

    # --- filter tasks ---
    if args.only:
        enabled_tasks = [t for t in enabled_tasks if t["id"] == args.only]
        if not enabled_tasks:
            print(f"ERROR: task {args.only} not found or not enabled.")
            return 1

    if args.from_task:
        found = False
        filtered = []
        for t in enabled_tasks:
            if t["id"] == args.from_task:
                found = True
            if found:
                filtered.append(t)
        if not found:
            print(f"ERROR: --from-task {args.from_task} not found in enabled tasks.")
            return 1
        enabled_tasks = filtered

    if args.max_tasks:
        enabled_tasks = enabled_tasks[: args.max_tasks]

    # --- dry-run ---
    if args.dry_run:
        print(f"ARCHIVED DRY-RUN: historical inventory contains {len(enabled_tasks)} task(s).")
        print("Execution is disabled; missing task_file paths are retained as historical evidence.")
        for t in enabled_tasks:
            print(f"  {t['id']}  {t['name']}")
            print(f"    branch: {t['branch']}")
            print(f"    task_file: {t['task_file']}")
            print(f"    commit: {t['commit_message']}")
        return 0

    # --- pre-flight checks ---
    if not check_branch(INTEGRATION_BRANCH):
        print(f"ERROR: must be on '{INTEGRATION_BRANCH}' branch. Currently on:")
        run(["git", "rev-parse", "--abbrev-ref", "HEAD"], check=False)
        return 1

    if not check_clean_worktree():
        print("ERROR: working tree is not clean. Please commit or stash changes first.")
        run(["git", "status", "--short"], check=False)
        return 1

    # --- agent commands ---
    dev_cmd = os.environ.get("DEV_AGENT_CMD", "claude -p")
    review_cmd = os.environ.get("REVIEW_AGENT_CMD", "claude -p")
    fix_cmd = os.environ.get("FIX_AGENT_CMD", "claude -p")

    print(f"DEV_AGENT_CMD   = {dev_cmd}")
    print(f"REVIEW_AGENT_CMD = {review_cmd}")
    print(f"FIX_AGENT_CMD   = {fix_cmd}")
    print(f"Tasks to run: {len(enabled_tasks)}")
    print(f"Args: only={args.only}, from_task={args.from_task}, max_tasks={args.max_tasks}, "
          f"no_merge={args.no_merge}, continue_on_failure={args.continue_on_failure}")
    print("-" * 60)

    if not args.yes:
        resp = input("Proceed? [y/N] ").strip().lower()
        if resp not in ("y", "yes"):
            print("Aborted.")
            return 0

    # --- ensure logs dir ---
    LOGS_DIR.mkdir(parents=True, exist_ok=True)

    # --- run each task ---
    stats = {"total": len(enabled_tasks), "passed": 0, "blocked": 0, "failed": 0, "skipped": 0}
    blocked_tasks: list[str] = []

    for idx, task in enumerate(enabled_tasks, start=1):
        task_id = task["id"]
        task_name = task["name"]
        task_file = task["task_file"]
        branch_name = task["branch"]
        commit_msg = task["commit_message"]
        max_fix = int(task.get("max_fix_rounds", 2))

        print(f"\n{'=' * 60}")
        print(f"[{idx}/{len(enabled_tasks)}] {task_id} — {task_name}")
        print(f"{'=' * 60}")

        status_entry: dict = {
            "task_id": task_id,
            "task_name": task_name,
            "branch": branch_name,
            "verdict": None,
            "fix_rounds": 0,
            "merged": False,
        }

        # --- 0. Check task file exists ---
        if not task_file_exists(task_file):
            print(f"  ERROR: task file not found: {task_file}")
            status_entry["verdict"] = "BLOCKED"
            status_entry["error"] = f"task_file_missing: {task_file}"
            write_status(status_entry)
            blocked_tasks.append(task_id)
            stats["blocked"] += 1
            if not args.continue_on_failure:
                _print_summary(stats, blocked_tasks)
                return 1
            continue

        # --- 1. Create branch ---
        step_checkout_integration()
        step_create_branch(branch_name)
        print(f"  Branch: {branch_name}")

        # --- 2. Developer Agent ---
        print(f"  [Developer] Running...")
        dev_vars = {
            "TASK_ID": task_id,
            "TASK_FILE": task_file,
            "COMMIT_MESSAGE": commit_msg,
        }
        dev_prompt = render_prompt(DEV_PROMPT_TEMPLATE, dev_vars)
        dev_output = call_agent(dev_prompt, dev_cmd)
        dev_log = log_file_path(task_id, "developer")
        dev_log.write_text(dev_output, encoding="utf-8")
        print(f"  [Developer] Output → {dev_log}")

        # --- 3. Verify commits ---
        # Check that the current branch has commits ahead of integration
        if not has_commits_ahead("HEAD", INTEGRATION_BRANCH):
            print(f"  ERROR: no commits on {branch_name} after Developer run.")
            status_entry["verdict"] = "BLOCKED"
            status_entry["error"] = "no_commits_produced"
            write_status(status_entry)
            blocked_tasks.append(task_id)
            stats["blocked"] += 1
            step_checkout_integration()
            run(["git", "branch", "-D", branch_name], check=False)
            if not args.continue_on_failure:
                _print_summary(stats, blocked_tasks)
                return 1
            continue

        # --- 4. Review loop ---
        verdict = None
        fix_round = 0

        while fix_round <= max_fix:
            review_round = fix_round + 1
            print(f"  [Reviewer] Round {review_round}...")

            review_vars = {"TASK_FILE": task_file}
            review_prompt = render_prompt(REVIEW_PROMPT_TEMPLATE, review_vars)
            review_output = call_agent(review_prompt, review_cmd)
            review_log = log_file_path(task_id, "review", review_round)
            review_log.write_text(review_output, encoding="utf-8")
            print(f"  [Reviewer] Output → {review_log}")

            verdict = parse_review_verdict(review_output)
            if verdict is None:
                # Could not parse — treat as BLOCKED (unexpected output)
                print(f"  ERROR: could not parse REVIEW_VERDICT from reviewer output.")
                verdict = "BLOCKED"

            status_entry["verdict"] = verdict
            status_entry["fix_rounds"] = fix_round

            if verdict == "PASS":
                print(f"  [Review] → PASS")
                break
            elif verdict == "BLOCKED":
                print(f"  [Review] → BLOCKED")
                break
            elif verdict == "NEEDS_FIX":
                if fix_round >= max_fix:
                    print(f"  [Review] → NEEDS_FIX but max fix rounds ({max_fix}) exhausted → BLOCKED")
                    verdict = "BLOCKED"
                    status_entry["verdict"] = "BLOCKED"
                    break

                print(f"  [Review] → NEEDS_FIX (round {fix_round + 1}/{max_fix})")
                print(f"  [Fixer] Running...")

                fix_vars = {
                    "TASK_ID": task_id,
                    "TASK_FILE": task_file,
                    "REVIEW_LOG": str(review_log),
                }
                fix_prompt = render_prompt(FIX_PROMPT_TEMPLATE, fix_vars)
                fix_output = call_agent(fix_prompt, fix_cmd)
                fix_log = log_file_path(task_id, "fix", fix_round + 1)
                fix_log.write_text(fix_output, encoding="utf-8")
                print(f"  [Fixer] Output → {fix_log}")

                # Verify fix produced commits
                if not has_commits_ahead("HEAD", INTEGRATION_BRANCH):
                    print(f"  WARNING: Fixer did not produce additional commits. Continuing review loop anyway.")

                fix_round += 1

        # --- 5. Handle final verdict ---
        if verdict == "PASS":
            if args.no_merge:
                print(f"  [Merge] SKIPPED (--no-merge). Branch {branch_name} left intact.")
                status_entry["merged"] = False
            else:
                try:
                    # Ensure we are on the task branch before merge
                    run(["git", "checkout", branch_name])
                    step_squash_merge(branch_name, commit_msg)
                    print(f"  [Merge] Squash-merged into {INTEGRATION_BRANCH}")
                    status_entry["merged"] = True
                    # Delete the task branch after successful merge
                    run(["git", "branch", "-D", branch_name], check=False)
                except RuntimeError as exc:
                    print(f"  [Merge] CONFLICT or ERROR: {exc}")
                    verdict = "BLOCKED"
                    status_entry["verdict"] = "BLOCKED"
                    status_entry["error"] = f"merge_failed: {exc}"
                    status_entry["merged"] = False
            stats["passed"] += 1
        else:
            # BLOCKED
            print(f"  [Task] BLOCKED — branch {branch_name} left for manual inspection.")
            blocked_tasks.append(task_id)
            stats["blocked"] += 1
            # Return to integration branch
            run(["git", "checkout", INTEGRATION_BRANCH], check=False)

        write_status(status_entry)

        if verdict != "PASS" and not args.continue_on_failure:
            _print_summary(stats, blocked_tasks)
            return 1

    _print_summary(stats, blocked_tasks)
    return 0 if stats["blocked"] == 0 else 1


def _print_summary(stats: dict, blocked_tasks: list[str]) -> None:
    print(f"\n{'=' * 60}")
    print("ORCHESTRATOR SUMMARY")
    print(f"  Total:   {stats['total']}")
    print(f"  Passed:  {stats['passed']}")
    print(f"  Blocked: {stats['blocked']}")
    print(f"  Failed:  {stats['failed']}")
    print(f"  Skipped: {stats['skipped']}")
    if blocked_tasks:
        print(f"  Blocked tasks: {', '.join(blocked_tasks)}")
    print(f"  Status log: {STATUS_FILE}")
    print(f"{'=' * 60}")


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------

def main() -> None:
    parser = argparse.ArgumentParser(
        description="Archived Serial Agent Orchestrator — inspect historical tasks only.",
    )
    parser.add_argument("--dry-run", action="store_true", help="Print the historical task inventory without executing.")
    parser.add_argument("--only", type=str, metavar="TASK_ID", help="Historical option; execution is disabled.")
    parser.add_argument("--from-task", type=str, metavar="TASK_ID", dest="from_task",
                        help="Historical option; execution is disabled.")
    parser.add_argument("--max-tasks", type=int, metavar="N", dest="max_tasks",
                        help="Historical option; execution is disabled.")
    parser.add_argument("--no-merge", action="store_true", dest="no_merge",
                        help="Historical option; execution is disabled.")
    parser.add_argument("--yes", action="store_true", help="Historical option; cannot bypass archive isolation.")
    parser.add_argument("--continue-on-failure", action="store_true", dest="continue_on_failure",
                        help="Historical option; execution is disabled.")

    args = parser.parse_args()

    # Mutual exclusion
    if args.only and args.from_task:
        print("ERROR: --only and --from-task are mutually exclusive.")
        sys.exit(1)

    sys.exit(orchestrate(args))


if __name__ == "__main__":
    main()
