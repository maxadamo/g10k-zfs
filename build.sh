#!/bin/bash
#
#
BIN_NAME=g10k-zfs
PATH=$PATH:$(go env GOPATH)/bin
GOPATH=$(go env GOPATH)
EMAIL="Massimiliano Adamo <maxadamo@gmail.com>"
DESC="leverage ZFS snapshots for G10K "
export BIN_NAME PATH GOPATH

git pull
LATEST_TAG=$(git describe --tags $(git rev-list --tags --max-count=1))
PROG_VERSION=$(echo $LATEST_TAG | sed -e 's/^v//')
BUILDTIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

rm -f ${BIN_NAME}
go get -d
git checkout $LATEST_TAG
go build -ldflags "-s -w -X main.appVersion=${PROG_VERSION} -X main.buildTime=${BUILDTIME}" -o ./${BIN_NAME}

if [ $? -gt 0 ]; then
    echo -e "\nthere was an error while compiling the code\n"
    exit
fi

echo -e "\nthe binary was compiled and it is avilable as:\n - $(pwd)/${BIN_NAME}\n"

# create RPM and DEB for amd64
if which fpm >/dev/null; then
    fpm -f -t rpm -n $BIN_NAME -v $PROG_VERSION --maintainer "$EMAIL" --vendor "$EMAIL" \
        -a x86_64 --description "$DESC" -s dir ./${BIN_NAME}=/usr/local/bin//${BIN_NAME}
    fpm -f -t deb -n $BIN_NAME -v ${PROG_VERSION}-1 --maintainer "$EMAIL" --vendor "$EMAIL" \
        -a amd64 --description "$DESC" -s dir ./${BIN_NAME}=/usr/local/bin/${BIN_NAME}
else
    echo -e "if you want to create RPM and DEB packages, please install fpm:\n - gem install fpm\n"
fi

git checkout master

