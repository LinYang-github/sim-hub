#!/bin/bash
# Check if http-server is installed, if not try npx
echo "Starting Demo Repository on http://localhost:30031..."
npx -y http-server ./apps/demo-repo -p 30031 --cors
