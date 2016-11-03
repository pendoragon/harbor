#!/bin/bash

# first parameter as version
version="${1:-latest}"

# script path
currentDir="$(pwd)"
scriptDir="$(cd $(dirname $0);pwd)"

# functions
# echoError(msg string)
echoError() {
    echo "error: $1"
}

# echoInfo(msg string)
echoInfo() {
    echo "info: $1"
}

# check(tag string) bool
check() {
    docker inspect $1 &>/dev/null
    if [ $? -eq 0 ]
    then
        echoInfo "$1 is already exists"
        return 1
    fi
    return 0
}

# buildImage(dockerfile string)
buildImage() {
    # check
    check harbor/$1:$version
    if [ $? -eq 1 ]
    then
        return
    fi
    echoInfo "start to build $1"
    docker build --tag harbor/$1:$version -f $scriptDir/$1.dockerfile .
    code=$?
    if [ $code -ne 0 ]
    then
        echoError "docker build failed with exit code:$code"
    else
        echoInfo "build $1 succeeded"
    fi
}

# pullImage(src string,name string)
pullImage() {
    # check
    check harbor/$2:$version
    if [ $? -eq 1 ]
    then
        return
    fi
    echoInfo "start to pull $1"
    docker pull $1
    code=$?
    if [ $code -ne 0 ]
    then
        echoError "docker pull failed with exit code:$code"
    else
        docker tag $1  harbor/$2:$version
        echoInfo "pull $1 succeeded"
    fi
}

# check docker
hash docker &>/dev/null
result=$?
if [ $result -ne 0 ]
then
    echoError "docker not found"
    exit 1
fi

# set build context:project root path
cd $scriptDir
cd ../../../

# build ui
buildImage "ui"

# build jobservice
buildImage "jobservice"

# build mysql
buildImage "mysql"


# end
cd $currentDir

# pull
if [ "$2" != "nopull" ]
then
    # pull registry
    pullImage "library/registry:2.5.0" "registry"
    # pull nginx
    pullImage "library/nginx:1.9" "nginx"
fi