#!/usr/bin/env bash
#
# Repackage every chart under assets/ with its version stripped of the
# "<anything>+up" prefix, e.g. fleet-108.0.0+up0.14.0.tgz -> fleet-0.14.0.tgz
#
set -euo pipefail

ASSETS_DIR="assets"
ENTRIES_DIR="entries"

# untar <tgz> <dest>
untar() {
	mkdir -p "$2"
	tar -xzf "$1" -C "$2" --strip-components=1
}

# strip_version <chart_yaml>
strip_version() {
	sed -i -E 's/^(version:[[:space:]]*).*\+up/\1/' "$1"
}

# repackage <src_dir> <dest_dir>
repackage() {
	helm package "$1" -d "$2"
}

main() {
	local tgz chart version dest tgzs

	# oldest -> newest, so on version collisions the newest repackage wins
	mapfile -t tgzs < <(printf '%s\n' "${ASSETS_DIR}"/*/*.tgz | sort -V)

	for tgz in "${tgzs[@]}"; do
		chart=$(basename "$(dirname "${tgz}")")  # assets/fleet/... -> fleet
		version=$(basename "${tgz}" .tgz)         # fleet-108.0.0+up0.14.0
		version=${version#"${chart}"-}            # 108.0.0+up0.14.0

		# already repackaged or no upstream version: nothing to strip
		[[ ${version} == *+up* ]] || continue

		dest="${ENTRIES_DIR}/${chart}/${version}"
		echo "==> ${chart} ${version}"

		untar "${tgz}" "${dest}"
		strip_version "${dest}/Chart.yaml"
		repackage "${dest}" "${ASSETS_DIR}/${chart}"
	done
}

main "$@"
