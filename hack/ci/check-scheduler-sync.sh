#!/usr/bin/env bash
#
# While the v1alpha scheduler exists, pkg/scheduling/v1alpha is a copy of
# pkg/scheduling/v1 with a rewritten scheduler core. Everything outside that
# core must be byte-for-byte identical between the two packages, modulo the
# package clause and the scheduling/v1[alpha] self-import path.
#
# This script enforces two rules:
#
#  1. Every file shared between the packages (i.e. not listed in DIVERGENT)
#     must be identical after normalization. To fix a failure, mirror the
#     change into the other package.
#
#  2. With --base <ref>, changes to a DIVERGENT (scheduler core) file in one
#     package must be accompanied by a change to the other package in the same
#     diff. The core files can't be compared mechanically, so this is a forcing
#     function: port the change, or make a deliberate no-op-explaining change
#     (e.g. a comment noting why the other implementation is unaffected).
#
# If a file becomes intentionally divergent (or a new core file is added),
# update DIVERGENT below.

set -euo pipefail

V1_DIR=pkg/scheduling/v1
ALPHA_DIR=pkg/scheduling/v1alpha

# The scheduler core that v1alpha rewrites (event loop + pool-owned slots),
# plus its tests and benchmarks.
DIVERGENT="
action.go
scheduler.go
scheduler_test.go
slot.go
slot_cost_test.go
slot_test.go
slot_pool.go
snapshot.go
scheduler_concurrent_bench_test.go
scheduler_shape_bench_test.go
scheduler_worker_scale_stress_bench_test.go
"

is_divergent() {
	grep -qx "$1" <<<"$DIVERGENT"
}

normalize() {
	sed -e 's/^package v1$/package SCHEDULING_IMPL/' \
		-e 's/^package v1alpha$/package SCHEDULING_IMPL/' \
		-e 's/^package v1_test$/package SCHEDULING_IMPL_test/' \
		-e 's/^package v1alpha_test$/package SCHEDULING_IMPL_test/' \
		-e 's|hatchet/pkg/scheduling/v1alpha"|hatchet/pkg/scheduling/SCHEDULING_IMPL"|' \
		-e 's|hatchet/pkg/scheduling/v1"|hatchet/pkg/scheduling/SCHEDULING_IMPL"|' \
		"$1"
}

fail=0

# --- Rule 1: shared files are identical after normalization -------------------

all_files=$( (cd "$V1_DIR" && ls ./*.go; cd - >/dev/null; cd "$ALPHA_DIR" && ls ./*.go) | xargs -n1 basename | sort -u)

for f in $all_files; do
	if is_divergent "$f"; then
		continue
	fi

	if [ ! -f "$V1_DIR/$f" ]; then
		echo "FAIL: $f exists in $ALPHA_DIR but not $V1_DIR — mirror it or add it to DIVERGENT in $0"
		fail=1
		continue
	fi

	if [ ! -f "$ALPHA_DIR/$f" ]; then
		echo "FAIL: $f exists in $V1_DIR but not $ALPHA_DIR — mirror it or add it to DIVERGENT in $0"
		fail=1
		continue
	fi

	if ! diff -u <(normalize "$V1_DIR/$f") <(normalize "$ALPHA_DIR/$f"); then
		echo "FAIL: $f differs between $V1_DIR and $ALPHA_DIR — the packages must stay in sync outside the scheduler core"
		fail=1
	fi
done

# --- Rule 2: core changes must touch both implementations ---------------------

if [ "${1:-}" = "--base" ]; then
	base="$2"
	changed=$(git diff --name-only "$base"...HEAD -- "$V1_DIR" "$ALPHA_DIR")

	v1_core_changed=0
	alpha_core_changed=0
	v1_changed=0
	alpha_changed=0

	for path in $changed; do
		f=$(basename "$path")
		case "$path" in
		"$V1_DIR"/*)
			v1_changed=1
			if is_divergent "$f"; then v1_core_changed=1; fi
			;;
		"$ALPHA_DIR"/*)
			alpha_changed=1
			if is_divergent "$f"; then alpha_core_changed=1; fi
			;;
		esac
	done

	if [ "$v1_core_changed" = 1 ] && [ "$alpha_changed" = 0 ]; then
		echo "FAIL: this diff changes the $V1_DIR scheduler core without touching $ALPHA_DIR."
		echo "Port the change to the v1alpha scheduler (or record why it does not apply)."
		fail=1
	fi

	if [ "$alpha_core_changed" = 1 ] && [ "$v1_changed" = 0 ]; then
		echo "FAIL: this diff changes the $ALPHA_DIR scheduler core without touching $V1_DIR."
		echo "Port the change to the v1 scheduler (or record why it does not apply)."
		fail=1
	fi
fi

if [ "$fail" = 1 ]; then
	exit 1
fi

echo "scheduler packages are in sync"
