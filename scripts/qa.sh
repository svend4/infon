#!/usr/bin/env bash
# qa.sh — one-command quality gate for the ray/dream stack, in the spirit of the
# `make qa` + smoke discipline of svend4/info150: build, vet, test, lint, a gofmt
# check of the actively-developed packages, then a headless smoke of the
# non-interactive commands. Prints a pass/fail summary and exits non-zero on any
# failure. Run from the repo root: scripts/qa.sh
set -uo pipefail
cd "$(dirname "$0")/.." || exit 2

ok=0
fail=0
log=$(mktemp)
step() { # step "name" cmd...
	local name="$1"
	shift
	if "$@" >"$log" 2>&1; then
		printf '  ok    %s\n' "$name"
		ok=$((ok + 1))
	else
		printf '  FAIL  %s\n' "$name"
		tail -8 "$log" | sed 's/^/        /'
		fail=$((fail + 1))
	fi
}

echo "== build / vet / test / lint =="
step "go build ./..." go build ./...
step "go vet ./..." go vet ./...
step "go test ./..." go test ./... -count=1
step "golangci-lint" golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 ./...
# gofmt only the packages this stack actively develops (pre-existing drift elsewhere
# is not gated).
if [ -z "$(gofmt -l pkg/raydir pkg/raytrace pkg/brain pkg/fleet 2>/dev/null)" ]; then
	printf '  ok    gofmt (raydir/raytrace/brain/fleet)\n'
	ok=$((ok + 1))
else
	printf '  FAIL  gofmt:\n'
	gofmt -l pkg/raydir pkg/raytrace pkg/brain pkg/fleet | sed 's/^/        /'
	fail=$((fail + 1))
fi

echo "== headless command smoke =="
tmp=$(mktemp -d)
step "rayvoxel" go run ./cmd/rayvoxel -w 96 -h 64 -out "$tmp/voxel.png"
step "rayfilm" go run ./cmd/rayfilm -frames 2 -cols 2 -fw 96 -fh 64 -regions 2 -out "$tmp/film.png"
step "conform (reference brain)" go run ./cmd/conform
step "rayfleet" go run ./cmd/rayfleet -w 96 -h 64 -spp 8 -vfx=false -out "$tmp/fleet"
step "raycamp" go run ./cmd/raycamp -gx 6 -gz 6 -w 96 -h 64 -spp 6 -out "$tmp/camp"
step "yijing_brain selftest" python3 ai/adapters/yijing_brain.py --selftest
step "equipment_brain selftest" python3 ai/adapters/equipment_brain.py --selftest

echo "== $ok ok, $fail fail =="
rm -f "$log"
rm -rf "$tmp"
[ "$fail" -eq 0 ]
