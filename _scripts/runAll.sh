#!/bin/sh

if [ $# -lt 2 ]; then
    echo "usage: ./runAll.sh outDir dir..."
    exit 1
fi

out=$1

rm -drf $out/*
for dir in $*; do
    if [ $dir = $1 ]; then
        continue
    fi
    echo $dir
    ./validateDir.sh $dir $out
done

rm -drf $out/*
for dir in $*; do
    if [ $dir = $1 ]; then
        continue
    fi
    echo $dir
    ./optimizeDir.sh $dir $out
done