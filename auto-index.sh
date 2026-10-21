#!/usr/bin/env bash
#
# Generate index.yaml from assets/ with every entry's urls pointing to its
# OCI reference, e.g. oci://zot.nicklab.org/charts/fleet:0.14.0
#
set -euo pipefail

ASSETS_DIR="assets"
REGISTRY="oci://zot.nicklab.org/charts"
INDEX="index.yaml"

main() {
	# helm always writes <dir>/index.yaml; its urls are rewritten below
	helm repo index "${ASSETS_DIR}" --url "${REGISTRY}"
	mv -f "${ASSETS_DIR}/${INDEX}" "${INDEX}"

	# <registry>/<chart>/<chart>-<version>.tgz -> <registry>/<chart>:<version>
	# sed, not yq: yq re-indents every sequence and the whole file shows in diff
	sed -i -E "s#(${REGISTRY})/([^/]+)/\2-(.+)\.tgz\$#\1/\2:\3#" "${INDEX}"
}

main "$@"
