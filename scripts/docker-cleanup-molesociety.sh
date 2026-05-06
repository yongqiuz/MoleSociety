#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DROP_DATA="${1:-}"

containers=(
  molesociety-frontend
  molesociety-backend
  molesociety-postgres
  molesociety-redis
  molesociety-backend-dev
  molesociety-postgres-dev
  molesociety-redis-dev
)

networks=(
  molesociety_default
)

volumes=(
  molesociety_postgres_data
  molesociety_redis_data
  molesociety_postgres_data_dev
  molesociety_redis_data_dev
)

images=(
  molesociety-backend
  molesociety-frontend
  molesociety_backend
  molesociety_frontend
)

remove_if_exists() {
  local kind="$1"
  local name="$2"
  if docker "$kind" inspect "$name" >/dev/null 2>&1; then
    echo "Removing $kind: $name"
    docker "$kind" rm -f "$name" >/dev/null
  fi
}

remove_network_if_exists() {
  local name="$1"
  if docker network inspect "$name" >/dev/null 2>&1; then
    echo "Removing network: $name"
    docker network rm "$name" >/dev/null || true
  fi
}

remove_volume_if_exists() {
  local name="$1"
  if docker volume inspect "$name" >/dev/null 2>&1; then
    echo "Removing volume: $name"
    docker volume rm -f "$name" >/dev/null
  fi
}

remove_image_if_exists() {
  local name="$1"
  if docker image inspect "$name" >/dev/null 2>&1; then
    echo "Removing image: $name"
    docker image rm -f "$name" >/dev/null || true
  fi
}

echo "Stopping compose stacks if present..."
if [ -f "$ROOT_DIR/docker-compose.prod.yml" ]; then
  if [ -f "$ROOT_DIR/.env.prod" ]; then
    docker compose --env-file "$ROOT_DIR/.env.prod" -f "$ROOT_DIR/docker-compose.prod.yml" down --remove-orphans >/dev/null 2>&1 || true
  else
    docker compose -f "$ROOT_DIR/docker-compose.prod.yml" down --remove-orphans >/dev/null 2>&1 || true
  fi
fi

if [ -f "$ROOT_DIR/docker-compose.dev.yml" ]; then
  if [ -f "$ROOT_DIR/.env.dev" ]; then
    docker compose --env-file "$ROOT_DIR/.env.dev" -f "$ROOT_DIR/docker-compose.dev.yml" down --remove-orphans >/dev/null 2>&1 || true
  else
    docker compose -f "$ROOT_DIR/docker-compose.dev.yml" down --remove-orphans >/dev/null 2>&1 || true
  fi
fi

echo "Removing known MoleSociety containers..."
for name in "${containers[@]}"; do
  remove_if_exists container "$name"
done

echo "Removing known MoleSociety networks..."
for name in "${networks[@]}"; do
  remove_network_if_exists "$name"
done

echo "Removing old MoleSociety images..."
for name in "${images[@]}"; do
  remove_image_if_exists "$name"
done

if [ "$DROP_DATA" = "--drop-data" ]; then
  echo "Removing MoleSociety data volumes..."
  for name in "${volumes[@]}"; do
    remove_volume_if_exists "$name"
  done
else
  echo "Keeping database and redis volumes. Pass --drop-data to delete them too."
fi

echo "Pruning dangling images only..."
docker image prune -f >/dev/null

echo "MoleSociety Docker cleanup finished."
