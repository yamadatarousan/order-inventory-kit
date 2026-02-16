#!/usr/bin/env bash
# 目的:
# DBマイグレーションの新規/変更スキーマに説明コメントがあることを機械検査する。
# 検査対象:
# - CREATE TABLE ... -> COMMENT ON TABLE
# - ALTER TABLE ... ADD COLUMN ... -> COMMENT ON COLUMN
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"
# ロケール依存の警告を避け、正規表現判定を安定化する。
export LC_ALL=C

has_error=0

for file in backend/migrations/*.up.sql; do
  [ -f "${file}" ] || continue

  # CREATE TABLE で追加されるテーブル名を抽出する。
  while IFS= read -r table; do
    [ -n "${table}" ] || continue
    if ! grep -Eiq "COMMENT ON[[:space:]]+TABLE[[:space:]]+${table}[[:space:]]+IS" "${file}"; then
      echo "[migration-comments] missing table comment: ${file} -> ${table}"
      has_error=1
    fi
  done <<EOF
$(perl -ne 'while(/CREATE TABLE(?: IF NOT EXISTS)?\s+([A-Za-z_][A-Za-z0-9_]*)/ig){print lc($1),"\n"}' "${file}" | sort -u)
EOF

  # ALTER TABLE ... ADD COLUMN ... で追加される列名を抽出する。
  while IFS= read -r pair; do
    [ -n "${pair}" ] || continue
    table="${pair%%:*}"
    column="${pair#*:}"
    if ! grep -Eiq "COMMENT ON[[:space:]]+COLUMN[[:space:]]+${table}\\.${column}[[:space:]]+IS" "${file}"; then
      echo "[migration-comments] missing column comment: ${file} -> ${table}.${column}"
      has_error=1
    fi
  done <<EOF
$(perl -0777 -ne '
  while(/ALTER TABLE\s+([A-Za-z_][A-Za-z0-9_]*)\s+(.*?);/sig){
    my $table = lc($1);
    my $body = $2;
    while($body =~ /ADD COLUMN(?:\s+IF NOT EXISTS)?\s+([A-Za-z_][A-Za-z0-9_]*)/ig){
      print $table, ":", lc($1), "\n";
    }
  }
' "${file}" | sort -u)
EOF
done

if [ "${has_error}" -ne 0 ]; then
  echo "[migration-comments] failed"
  exit 1
fi

echo "[migration-comments] ok"
