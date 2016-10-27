#! /bin/sh

#env
version="0.4.0"

#script path
currnt_dir="$(pwd)"
script_dir="$(cd $(dirname ${0});pwd)"

#functions
#EchoError(msg string)
EchoError() {
    echo "\033[31m$1\033[0m"
}

#EchoInfo(msg string)
EchoInfo() {
    echo "\033[33m$1\033[0m"
}

#Check(tag string) bool
Check() {
    docker inspect $1 &>/dev/null
    if [ $? -eq 0 ]
    then
        EchoInfo "$1 is already exists"
        return 1
    fi
    return 0
}

#BuildImage(dockerfile string)
BuildImage() {
    #check
    Check harbor/$1:${version}
    if [ $? -eq 1 ]
    then
        return
    fi
    EchoInfo "start to build $1"
    docker build --tag harbor/$1:${version} -f ${script_dir}/$1.dockerfile .
    code=$?
    if [ ${code} -ne 0 ]
    then
        EchoError "docker build failed with exit code:${code}"
    else
        EchoInfo "build $1 succeeded"
    fi
}

#PullImage(src string,name string)
PullImage() {
    #check
    Check harbor/$2:${version}
    if [ $? -eq 1 ]
    then
        return
    fi
    EchoInfo "start to pull $1"
    docker pull $1
    code=$?
    if [ ${code} -ne 0 ]
    then
        EchoError "docker pull failed with exit code:${code}"
    else
        docker tag $1  harbor/$2:${version}
        EchoInfo "pull $1 succeeded"
    fi
}

#check docker
hash docker &>/dev/null
result=$?
if [ ${result} -ne 0 ]
then
    EchoError "docker not found"
    exit 1
fi

#set build context:project root path
cd ${script_dir}
cd ../../../

#build ui
BuildImage "ui"

#build jobservice
BuildImage "jobservice"

#build mysql
BuildImage "mysql"


#end
cd ${currnt_dir}

# pull
if [ "$1" != "nopull" ]
then
    #pull registry
    PullImage "library/registry:2.5.0" "registry"
    #pull nginx
    PullImage "library/nginx:1.9" "nginx"
fi