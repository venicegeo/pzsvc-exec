#!/bin/bash

export PATH="$HOME/miniconda2/bin:$PATH"

pwd
ls
rm -rf miniconda2/conda-bld
cd ./share
ls vendor/pzsvc-exec
conda build pzsvc-exec
cd
mkdir linux-64 && cd linux-64
wget -r -l1 -e robots=off -nH -nd --reject="index.html*" --no-parent --no-cookies https://nexus.devops.geointservices.io/content/repositories/beachfront-conda/linux-64/ --user=proxy --password=proxy
rm pzsvc-exec*
cd ..
mv miniconda2/conda-bld/linux-64/pzsvc-exec* linux-64/
conda index linux-64
cd linux-64
find . -type f ! -name 'pzsvc-exec*' ! -name 'repodata.*' -delete
ls
cd
mv linux-64 share/
