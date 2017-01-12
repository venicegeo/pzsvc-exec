#! /bin/bash -ex

pushd `dirname $0`/.. > /dev/null
root=$(pwd -P)
popd > /dev/null

export GOPATH=$root/gopath

source $root/ci/vars.sh

mkdir -p $GOPATH $GOPATH/bin $GOPATH/src $GOPATH/pkg

PATH=$PATH:$GOPATH/bin

go version

# install metalinter
go get -u github.com/alecthomas/gometalinter
gometalinter --install

go get -v github.com/venicegeo/pzsvc-exec/...

# for pzse library: unit test w/ code coverage, then lint
cd $GOPATH/src/github.com/venicegeo/pzsvc-exec/pzse
go test -v -coverprofile=$root/pzse.cov github.com/venicegeo/pzsvc-exec/pzse

gometalinter \
--deadline=60s \
--concurrency=6 \
--vendor \
--exclude="exported (var)|(method)|(const)|(type)|(function) [A-Za-z\.0-9]* should have comment" \
--exclude="comment on exported function [A-Za-z\.0-9]* should be of the form" \
--exclude="Api.* should be .*API" \
--exclude="Http.* should be .*HTTP" \
--exclude="Id.* should be .*ID" \
--exclude="Json.* should be .*JSON" \
--exclude="Url.* should be .*URL" \
--exclude="[iI][dD] can be fmt\.Stringer" \
--exclude=" duplicate of [A-Za-z\._0-9]*" \
./... | tee $root/pzse-lint.txt
wc -l $root/pzse-lint.txt


# for pzsvc library: unit test w/ code coverage, then lint
cd $GOPATH/src/github.com/venicegeo/pzsvc-exec/pzsvc
go test -v -coverprofile=$root/pzsvc.cov github.com/venicegeo/pzsvc-exec/pzsvc

gometalinter \
--deadline=60s \
--concurrency=6 \
--vendor \
--exclude="exported (var)|(method)|(const)|(type)|(function) [A-Za-z\.0-9]* should have comment" \
--exclude="comment on exported function [A-Za-z\.0-9]* should be of the form" \
--exclude="Api.* should be .*API" \
--exclude="Http.* should be .*HTTP" \
--exclude="Id.* should be .*ID" \
--exclude="Json.* should be .*JSON" \
--exclude="Url.* should be .*URL" \
--exclude="[iI][dD] can be fmt\.Stringer" \
--exclude=" duplicate of [A-Za-z\._0-9]*" \
./... | tee $root/pzsvc-lint.txt
wc -l $root/pzsvc-lint.txt

# for taskworker, install appropriately
cd $GOPATH/src/github.com/venicegeo/pzsvc-exec/pzsvc-taskworker
go install .

cd $root
cp $GOPATH/bin/$APP ./$APP.bin
cp $GOPATH/bin/pzsvc-taskworker ./pzsvc-taskworker.bin
tar cvzf $APP.$EXT \
    $APP.bin \
    pzsvc-taskworker.bin \
    pzsvc.cov \
    pzse.cov \
    pzse-lint.txt \
    pzsvc-lint.txt
tar tzf $APP.$EXT

