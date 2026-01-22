#!/bin/bash
# Check if http-server is installed, if not try npx
echo "Starting External Demo Repository on http://localhost:30031..."
npx -y http-server ./external-demo -p 30031 --cors
