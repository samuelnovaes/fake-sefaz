#!/usr/bin/env bash
set -euo pipefail

# Downloads the published XSD packages into the directory given as the first
# argument, defaulting to ./schemas. The packages are the ones the SEFAZ
# publishes on the Portal da NF-e; this script pulls the copies kept by the
# open source projects that track them, so the download needs no browser.

target="${1:-schemas}"
mkdir -p "$target"

fetch() {
  local repository="$1" path="$2" folder="$3"
  echo "==> $folder"
  mkdir -p "$target/$folder"
  curl -fsS "https://api.github.com/repos/$repository/contents/$path" \
    | grep -o '"download_url": *"[^"]*\.xsd"' \
    | sed 's/.*"\(https[^"]*\)"/\1/' \
    | (cd "$target/$folder" && xargs -P 8 -n 1 curl -fsS -O)
  echo "    $(find "$target/$folder" -name '*.xsd' | wc -l) schemas"
}

fetch nfephp-org/sped-nfe schemes/PL_009_V4 nfe
fetch nfephp-org/sped-cte schemes/PL_CTe_400 cte
fetch nfephp-org/sped-mdfe schemes/PL_MDFe_300a mdfe

echo
echo "Point the service at them with:"
echo "  FAKE_SEFAZ_SCHEMA_DIR=$target go run ./cmd/fakesefaz"
