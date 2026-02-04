#!/usr/bin/env bash
set -euo pipefail

# TODO: Generate Go/TS types and clients from contracts/openapi.yaml
# Example:
# oapi-codegen -generate types,gin -o backend/internal/adapter/http/openapi.gen.go contracts/openapi.yaml
# openapi-typescript contracts/openapi.yaml --output frontend/src/api/schema.d.ts

echo "[openapi-generate] placeholder: no generator configured"
