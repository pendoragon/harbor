#!/bin/bash

# exit immediately if a command exits with a non-zero status.
set -e

echo "This shell will launch Harbor project on local with cargo config"
echo "Usage: ./depoly-cargo.sh"

# ANSI escape codes
# Black        0;30     Dark Gray     1;30
# Red          0;31     Light Red     1;31
# Green        0;32     Light Green   1;32
# Brown/Orange 0;33     Yellow        1;33
# Blue         0;34     Light Blue    1;34
# Purple       0;35     Light Purple  1;35
# Cyan         0;36     Light Cyan    1;36
# Light Gray   0;37     White         1;37

RED='\033[0;31m' # red color
NC='\033[0m' # no color

# detect the platform
# ref:
# http://stackoverflow.com/a/394247/3167471

platform='unknown'
unamestr=`uname -s`

if [[ "$unamestr" == 'Linux' ]]; then
   platform='linux'
elif [[ "$unamestr" == 'Darwin' ]]; then
   platform='mac'
else
    echo -e "${RED}Error: platform: $platform${NC}"
    exit 1
fi

echo "platform: $platform"


# detect ip address
# ref:
# http://stackoverflow.com/a/23934900/3167471

ip='unknown'

if [[ "$platform" == 'linux' ]]; then
   ip=`ip addr show eth0 | awk '$1 == "inet" {gsub(/\/.*$/, "", $2); print $2}'`
elif [[ "$platform" == 'mac' ]]; then
   ip=`ifconfig en0 | awk '$1 == "inet" {print $2}'`
fi

echo "ip: $ip"

hostname="$ip:8002"

# ref;
# Most often single quotes are used,
# to avoid having the shell interpret $ as a shell variable.
# Double quotes are used, such as "s/$1/$2/g",
# to allow the shell to substitute for a command line argument or other shell variable.

echo "config hostname to $hostname..."
sed "s/hostname =.*$/hostname = $hostname/g" -i ./harbor.cfg

echo "prepare config..."
./prepare

echo "nginx listen on 8002..."
sed 's/listen.*;$/listen 8002;/g' -i ./config/nginx/nginx.conf

echo "build gobase for harbor/ui and harbor/job..."
docker build -f ../Dockerfile.gobase -t gobase:latest ../

echo "docker-compose up cargo..."
docker-compose -f docker-compose-cargo.yml up -d
