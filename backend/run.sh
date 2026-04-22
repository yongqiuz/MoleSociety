#!/usr/bin/env bash
set -euo pipefail

APP_NAME="molesociety-backend"
LOG_FILE="backend.log"
JAR_FILE="target/${APP_NAME}-0.0.1-SNAPSHOT.jar"

echo "Starting MoleSociety Spring Boot backend..."

if ! command -v mvn >/dev/null 2>&1; then
  echo "Maven is required but was not found in PATH."
  exit 1
fi

if ! command -v java >/dev/null 2>&1; then
  echo "Java 17+ is required but was not found in PATH."
  exit 1
fi

PID="$(pgrep -f "${JAR_FILE}" || true)"
if [ -n "${PID}" ]; then
  echo "Stopping existing backend process: ${PID}"
  kill ${PID}
fi

mvn -q -DskipTests package

nohup java -jar "${JAR_FILE}" > "${LOG_FILE}" 2>&1 &
NEW_PID=$!

echo "Backend started."
echo "PID: ${NEW_PID}"
echo "Log: ${LOG_FILE}"
echo "Health: http://127.0.0.1:8080/healthz"
