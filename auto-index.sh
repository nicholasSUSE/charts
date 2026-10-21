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
	helm repo index "${ASSETS_DIR}"
	mv -f "${ASSETS_DIR}/${INDEX}" "${INDEX}"

	# <path>/<chart>-<version>.tgz -> <registry>/<chart>:<version>
	REGISTRY="${REGISTRY}" yq -i \
		'.entries[][] |= .urls = [strenv(REGISTRY) + "/" + .name + ":" + .version]' \
		"${INDEX}"
}

main "$@"
