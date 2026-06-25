#!/usr/bin/env sh
# Build (if needed) and run the menu-bar app.
cd "$(dirname "$0")"
go build -o charging-app . || exit 1
exec ./charging-app
