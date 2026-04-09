@echo off
setlocal

cd /d "%~dp0"

echo Whale Vault dev environment
echo.
echo This window will run frontend and backend together.
echo Closing this window will stop both services.
echo.

for %%P in (4173 8080) do (
  powershell -NoProfile -Command "if (Get-NetTCPConnection -LocalPort %%P -State Listen -ErrorAction SilentlyContinue) { exit 1 }"
  if errorlevel 1 (
    echo Port %%P is already in use. Please stop the existing process first.
    exit /b 1
  )
)

npx concurrently ^
  --names "backend,frontend" ^
  --prefix "[{name}]" ^
  --prefix-colors "magenta,cyan" ^
  --kill-others ^
  --kill-others-on-fail ^
  "npm run dev:backend" ^
  "npm run dev:frontend"
