#!/usr/bin/env bash
# Prints the claims of the OIDC token of this job, and the values the Vault
# login of rancher-eio/read-vault-secrets would use. The token itself is never
# printed.
set -euo pipefail

label=${1:?label}

token=$(curl -fsSL -H "Authorization: bearer ${ACTIONS_ID_TOKEN_REQUEST_TOKEN}" "${ACTIONS_ID_TOKEN_REQUEST_URL}" | jq -r .value)
echo "::add-mask::${token}"

payload=$(printf '%s' "$token" | cut -d. -f2 | tr '_-' '/+')
# base64url has no padding
while [ $(( ${#payload} % 4 )) -ne 0 ]; do payload="${payload}="; done

echo "### ${label}" | tee -a "$GITHUB_STEP_SUMMARY"
{
  echo '```'
  printf '%s' "$payload" | base64 -d | jq '{sub, repository, repository_owner, ref, event_name, workflow, workflow_ref, job_workflow_ref, runner_environment}'
  echo "github.repository_owner  (vault auth path): ${GH_OWNER}"
  echo "github.event.repository.name (vault role):  ${GH_REPO_NAME}"
  echo "github.repository: ${GH_REPOSITORY}"
  echo '```'
} | tee -a "$GITHUB_STEP_SUMMARY"
