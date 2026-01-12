#!/bin/bash


# Script to install the Hatchet CLI on macOS, Linux, and WSL


osname=""
version=""


check_prereqs() {
    command -v curl >/dev/null 2>&1 || { echo "[ERROR] curl is required to install the Hatchet CLI." >&2; exit 1; }
    command -v unzip >/dev/null 2>&1 || { echo "[ERROR] unzip is required to install the Hatchet CLI." >&2; exit 1; }
}


download_and_install() {
    curl -s -L https://github.com/hatchet-dev/hatchet/releases/download/${version}/$1_${version}_${osname}_x86_64.zip --output $1.zip
    unzip -a $1.zip
    rm $1.zip


    chmod +x ./$1
    sudo mv ./$1 /usr/local/bin/$1


    command -v $1 >/dev/null 2>&1 || { echo "[ERROR] There was an error installing the Hatchet CLI. Please try again." >&2; exit 1; }
}


download_and_install_all() {
    check_prereqs


    version=$(curl --silent "https://api.github.com/repos/hatchet-dev/hatchet/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

    echo "[INFO] Since the Hatchet CLI gets installed in /usr/local/bin, you may be asked to input your password."
    echo "[INFO] Please make sure /usr/local/bin is included in your PATH."


    download_and_install hatchet-server
    download_and_install hatchet-admin
    download_and_install hatchet


    exit
}


if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    if uname -a | grep -q '^Linux.*Microsoft'; then
        echo "[WARNING] WSL support is experimental and may result in crashes."
    fi
    osname="Linux"
    download_and_install_all
elif [[ "$OSTYPE" == "darwin"* ]]; then
    osname="Darwin"
    download_and_install_all
fi


echo "[ERROR] Unsupported operating system."
exit 1
