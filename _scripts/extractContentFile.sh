#!/bin/sh

#: ./extractContentFile.sh ~/pdf/1mb/a.pdf ~/pdf/out

if [ $# -ne 2 ]; then
    echo "usage: ./extractContentFile.sh inFile outDir"
    exit 1
fi

f=${1##*/}
f1=${f%.*}
out=$2

#rm -drf $out/*

#set -e

mkdir $out/$f1
cp $1 $out/$f1 

pdfcpu extract -verbose -mode=content $out/$f1/$f $out/$f1 &> $out/$f1/$f1.log
if [ $? -eq 1 ]; then
    echo "content extraction error: $1 -> $out/$f1"
    exit $?
else
    echo "content extraction success: $1 -> $out/$f1"
fi
	

	
