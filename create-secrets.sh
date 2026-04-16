#!/bin/bash

# All secrets are read from .env.secrets (not version-controlled).
# Required keys:
#   GHCR_USERNAME  — GitHub username for container registry
#   GHCR_PAT       — GitHub personal access token (read:packages scope)
#   GHCR_EMAIL     — GitHub account email
#   BOT_TOKEN      — Telegram bot token
#   DB_URL         — full postgres connection string (e.g. postgres://user:pass@postgres-svc:5432/bot?sslmode=disable)
#   DB_PASSWORD    — postgres password (must match DB_URL; used by the postgres StatefulSet)

set -o allexport
source .env.secrets
set +o allexport

# Registry auth
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username="${GHCR_USERNAME}" \
  --docker-password="${GHCR_PAT}" \
  --docker-email="${GHCR_EMAIL}" \
  --dry-run=client -o yaml | kubectl apply -f -

# App secrets
kubectl create secret generic app-secrets \
  --from-literal=BOT_TOKEN="${BOT_TOKEN}" \
  --from-literal=DB_URL="${DB_URL}" \
  --from-literal=DB_PASSWORD="${DB_PASSWORD}" \
  --dry-run=client -o yaml | kubectl apply -f -
