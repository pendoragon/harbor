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
    if [ $? -eq 0 ];then
        echoInfo "$1 is already exists"
        return 1
    fi
    return 0
}

# buildImage(dockerfile string)
buildImage() {
    # check
    check harbor/$1:$version
    if [ $? -eq 1 ];then
        return
    fi
    echoInfo "start to build $1"
    docker build --tag harbor/$1:$version -f $scriptDir/$1.dockerfile .
    code=$?
    if [ $code -ne 0 ];then
        echoError "docker failed to build $1 with exit code:$code"
        exit 1
    else
        echoInfo "build $1 succeeded"
    fi
}

# pullImage(src string,name string)
pullImage() {
    # check
    check harbor/$2:$version
    if [ $? -eq 1 ];then
        return
    fi
    echoInfo "start to pull $1"
    docker pull $1
    code=$?
    if [ $code -ne 0 ];then
        echoError "docker failed to pull $1 with exit code:$code"
    else
        docker tag $1  harbor/$2:$version
        echoInfo "pull $1 succeeded"
    fi
}

# check docker
hash docker &>/dev/null
result=$?
if [ $result -ne 0 ];then
    echoError "docker not found"
    exit 1
fi

# set build context:project root path
cd $scriptDir
cd ../../../

# build base image
buildImage "build"

echoInfo "build ui & jobservice"
if [ ! -d ./bin ];then
    mkdir bin
    if [ $? -ne 0 ];then
        echoError "mkdir failed to create $scriptDir/bin"
        exit 1
    fi
fi
docker run -v $scriptDir/bin:/dist  harbor/build:$version cp -rf /go/bin/. /dist/
if [ $? -ne 0 ];then
    echoError "docker failed to run harbor/build with exit code:$code"
    exit 1
fi


# build ui
buildImage "ui"

# build jobservice
buildImage "jobservice"

# build mysql
buildImage "mysql"

# clean
docker rm `docker ps -f status=exited -f ancestor=harbor/build:$version -q`
if [ $? -ne 0 ];then
    echoError "docker failed to remove container harbor/build:$version with exit code:$code"
fi
docker rmi harbor/build:$version
if [ $? -ne 0 ];then
    echoError "docker failed to remove image harbor/build:$version with exit code:$code"
fi
rm -rf $scriptDir/bin
if [ $? -ne 0 ];then
    echoError "failed clean $scriptDir/bin"
fi

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


