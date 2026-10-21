#!/usr/bin/env bash
#
# Push every repackaged chart (no "+up" in its version) under assets/ to the
# OCI registry.
#
set -euo pipefail

ASSETS_DIR="assets"
REGISTRY="oci://zot.nicklab.org/charts"

main() {
	local tgz failed=()

	for tgz in "${ASSETS_DIR}"/*/*.tgz; do
		# originals (<rancher>+up<upstream>) are not pushed, only repackaged ones
		[[ ${tgz} == *+up* ]] && continue

		echo "==> ${tgz}"
		helm push "${tgz}" "${REGISTRY}" --plain-http || failed+=("${tgz}")
	done

	if (( ${#failed[@]} )); then
		printf 'FAILED: %s\n' "${failed[@]}" >&2
		exit 1
	fi
}

main "$@"
