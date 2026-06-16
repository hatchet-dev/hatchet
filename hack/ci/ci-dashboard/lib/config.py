"""Static configuration for the CI health dashboard tool."""

from pathlib import Path

REPO = "hatchet-dev/hatchet"
WINDOW_DAYS = 14
TABLE_WINDOW_HOURS = 24
TOP_N = 10

BASE_DIR = Path(__file__).resolve().parents[1]  # hack/ci/ci-dashboard/
CACHE_DIR = BASE_DIR / ".cache"
OUT_DIR = BASE_DIR / "out"

RUNS_DIR = CACHE_DIR / "runs"
JOBS_DIR = CACHE_DIR / "jobs"  # reserved; jobs are embedded in run files
JOB_FAILURES_DIR = CACHE_DIR / "job-failures"
CLASSIFICATIONS_FILE = CACHE_DIR / "classifications.json"
META_FILE = CACHE_DIR / "meta.json"

ANALYSIS_FILE = OUT_DIR / "analysis.json"
ISSUE_FILE = OUT_DIR / "issue.md"

# The canonical dashboard issue. publish.py updates this exact issue in place
# (override per-run with `publish.py --issue <n>`).
DASHBOARD_ISSUE = 4204
DASHBOARD_TITLE = "CI Health Dashboard"
DASHBOARD_LABEL = "ci-health"

# Workflows whose jobs/tests are ranked + classified (PR/main-gating CI).
# Every workflow's run-level data is still collected; this only limits which
# workflows feed the top-N job/test ranking and the pass-rate totals.
GATING_WORKFLOWS = {
    "test",
    "typescript",
    "python",
    "go",
    "ruby",
    "cli-e2e-tests",
    "lint all",
    "build",
    "e2e-image",
    "Template E2E Tests",
    "frontend / app",
    "frontend-app-cypress",
    "frontend / docs",
}

CATEGORIES = [
    "product bug",
    "flaky test",
    "infra/CI",
    "dependency",
    "timeout",
    "data/env",
    "unknown",
]
