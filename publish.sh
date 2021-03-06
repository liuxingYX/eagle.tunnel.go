#!/bin/bash
###
# @Author: EagleXiang
# @LastEditors: EagleXiang
# @Email: eagle.xiang@outlook.com
# @Github: https://github.com/eaglexiang
# @Date: 2019-08-24 16:56:37
# @LastEditTime: 2019-08-28 21:41:55
###

defaultOS=linux
defaultArch=amd64

get_os() {
    # $1:os
    if [ "$1" ]; then
        os=$1
    else
        os=$defaultOS
    fi
    
    if [ $os = "mac" ]; then
        os="darwin"
    fi
    
    if [ X$os == "X" ]; then
        echo "no os"
        exit 1
    fi
}

get_arch() {
    # $1:arch
    if [ "$1" ]; then
        arch=$1
    else
        arch=$defaultArch
    fi
    
    if [ X$arch == "X" ]; then
        echo "no arch"
        exit 1
    fi
}

get_bin_name() {
    # suffix
    if [ $os == "windows" ]; then
        suffix="exe"
    else
        suffix=$os
    fi
    
    bin="et.go.$suffix"
}

get_release_name() {
    release="et.go.$os.$arch"
    
    if [ $os == "windows" ]; then
        release="${release}.zip"
    else
        release="${release}.tar.gz"
    fi
}

compress() {
    # release folder
    echo "PREPARE TEMPORARY FOLDER"
    release_folder="EagleTunnel"
    
    mkdir -p $release_folder
    \cp -f $bin $release_folder
    # config
    release_config=$release_folder/config
    mkdir -p $release_config
    \cp -rf ./config/* $release_config
    # scripts
    \cp -f ./install_linux.sh $release_folder/
    \cp -f ./uninstall_linux.sh $release_folder/
    
    echo -e "COMPRESS:\t$release"
    if [ $os == "windows" ];then
        zip -r $release $release_folder
    else
        tar -czvf $release $release_folder
    fi
    
    if [ -f $release ]; then
        echo "COMPRESS DONE"
        echo "============================================================"
    else
        echo "COMPRESS ERROR"
        echo "============================================================"
        exit 1
    fi
}

clean() {
    if [ -f $bin ]; then
        rm -f $bin
        if [ ! -f $bin ]; then
            echo -e "REMOVED:\t$bin"
        fi
        rm -rf $release_folder
        if [ ! -d $release_folder ]; then
            echo -e "REMOVED:\t$release_folder"
        fi
        echo "============================================================"
    fi
}

publish() {
    get_os "$1"
    get_arch "$2"
    get_bin_name
    get_release_name
    
    ./build.sh "$1" "$2"
    compress
    
    clean
    
    mkdir -p ./zip
    mv $release ./zip/
}

if [ "$1" ] && [ "$1" == "clean" ]
then
    rm -rf ./zip
    echo ""
    exit 0
fi

# update clear domains
./update_clear_domains.sh
echo ""

if [ "$1" ] && [ "$1" == "all" ]
then
    publish linux amd64
    echo ""
    publish linux 386
    echo ""
    publish linux arm
    echo ""
    publish linux arm64
    echo ""
    publish windows amd64
    echo ""
    publish windows 386
    echo ""
    publish darwin amd64
    echo ""
    publish darwin 386
    echo ""
    echo "RELEASES PUBLISH FINISHED"
else
    publish "$1" "$2"
fi
