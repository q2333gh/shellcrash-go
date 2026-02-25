#!/bin/sh
set -eu

echo "***********************************************"
echo "**               Welcome to                  **"
echo "**       ShellCrash Go Installer             **"
echo "**                             by  Juewuy    **"
echo "***********************************************"

echo "This install script has been migrated to a pure Go entrypoint: shellcrash-installctl."
echo
echo "Please use one of the following (recommended 1 or 2):"
echo
echo " 1) From a packaged release:"
echo "      shellcrash-installctl"
echo
echo " 2) From source with Go toolchain:"
echo "      cd shellcrashgo"
echo "      go run ./cmd/installctl"
echo
echo "For custom install directory or other options, see:"
echo "      shellcrash-installctl -h"
echo
echo "To satisfy the pure Go requirement, this install_en.sh no longer performs any install logic."

exit 0

