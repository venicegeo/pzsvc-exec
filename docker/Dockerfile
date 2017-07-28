FROM centos:centos7

WORKDIR /root

RUN yum -y install epel-release && yum -y update &&\
    yum -y install zlib-devel gcc gcc-c++ autogen libtool make wget autoconf\
    automake freetype-devel SDL-devel python-devel python-pip bzip2 swig\
    openssl-devel &&\
    curl -L https://repo.continuum.io/miniconda/Miniconda2-latest-Linux-x86_64.sh -o miniconda.sh &&\
    /usr/bin/bash miniconda.sh -b -p $HOME/miniconda2 &&\
    rm miniconda.sh &&\
    export PATH="$HOME/miniconda2/bin:$PATH" &&\
    conda install conda-build

ADD setup.sh /root/

CMD chmod 755 setup.sh && ./setup.sh
