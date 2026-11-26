#!/usr/bin/env bash

# Installs a Go tool with `go install`, unless it is already installed at the requested version.
#
# Usage: install-tool.sh <package>@<version> [go install flags...]
#
# The installed version is read from the build info embedded in the binary (`go version -m`). For
# `@latest`, the binary is reinstalled only when a newer module version is available. Set
# FORCE_INSTALL_TOOLS=true to always reinstall.

set -Eeuo pipefail

package_version="${1:?Provide <package>@<version> as arg[1]}"
shift

package="${package_version%@*}"
version="${package_version##*@}"
binary="$(basename "${package}")"
# Major version suffixes are not part of the binary name (e.g. github.com/mikefarah/yq/v3 -> yq).
if [[ "${binary}" =~ ^v[0-9]+$ ]]; then
  binary="$(basename "$(dirname "${package}")")"
fi

function install_tool {
  echo "Installing ${package_version}"
  GOFLAGS='' go install "$@" "${package_version}"
}

if [[ "${FORCE_INSTALL_TOOLS:-false}" == "true" ]]; then
  install_tool "$@"
  exit 0
fi

gobin="$(go env GOBIN)"
gobin="${gobin:-$(go env GOPATH)/bin}"

# Find an installed binary that was built from the requested package. Prefer the `go install`
# target directory, then fall back to PATH (e.g. tools preinstalled in a CI image).
installed_path=""
installed_module=""
installed_version=""
for candidate in "${gobin}/${binary}" "$(type -P "${binary}" || true)"; do
  [[ -n "${candidate}" && -x "${candidate}" ]] || continue
  build_info="$(go version -m "${candidate}" 2>/dev/null || true)"
  if [[ "$(awk '$1 == "path" { print $2 }' <<< "${build_info}")" == "${package}" ]]; then
    installed_path="${candidate}"
    installed_module="$(awk '$1 == "mod" { print $2 }' <<< "${build_info}")"
    installed_version="$(awk '$1 == "mod" { print $3 }' <<< "${build_info}")"
    break
  fi
done

if [[ -z "${installed_path}" ]]; then
  install_tool "$@"
  exit 0
fi

wanted_version="${version}"
if [[ "${version}" == "latest" ]]; then
  # Query outside of any module, so that the repository's go.mod (vendoring, replaces) doesn't interfere.
  if ! wanted_version="$(cd "$(mktemp -d)" && GOFLAGS='' go list -m -f '{{.Version}}' "${installed_module}@latest" 2>/dev/null)"; then
    echo "${binary} ${installed_version} is installed, unable to check for the latest version, keeping it"
    exit 0
  fi
fi

if [[ "${installed_version}" == "${wanted_version}" ]]; then
  echo "${binary} ${installed_version} is already installed at ${installed_path}"
  exit 0
fi

echo "${binary} ${installed_version} is installed at ${installed_path}, but ${wanted_version} is requested"
install_tool "$@"
