#!/bin/sh
# Detect distro family from /etc/os-release
. /etc/os-release 2>/dev/null

for id in $ID $ID_LIKE; do
  case $id in
    debian|ubuntu|linuxmint|pop) echo deb; exit 0;;
    rhel|centos|fedora|rocky|almalinux|opensuse-leap|opensuse-tumbleweed) echo rpm; exit 0;;
  esac
done

echo unknown
