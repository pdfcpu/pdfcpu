#!/bin/sh

#: ./validateFile.sh ~/pdf/1mb/a.pdf ~/pdf/out

if [ $# -ne 2 ]; then
    echo "usage: ./validateFile.sh inFile logDir"
    exit 1
fi

f=${1##*/}
f1=${f%.*}
out=$2

#rm -drf $out/*

cp $1 $out/$f

pdfcpu validate -verbose -mode=relaxed $out/$f &> $out/$f1.log

if [ $? -eq 1 ]; then
    echo "validation error: $out/$f"
    exit $?
else
    echo "validation success: $out/$f"
fi