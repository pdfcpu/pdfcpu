#!/bin/sh

#: ./extractFontsFile.sh ~/pdf/1mb/a.pdf ~/pdf/out

if [ $# -ne 2 ]; then
    echo "usage: ./extractFontsFile.sh inFile outDir"
    exit 1
fi

f=${1##*/}
f1=${f%.*}
out=$2

#rm -drf $out/*

#set -e

mkdir $out/$f1
cp $1 $out/$f1 

pdfcpu extract -verbose -mode=font $out/$f1/$f $out/$f1 &> $out/$f1/$f1.log
if [ $? -eq 1 ]; then
    echo "font extraction error: $1 -> $out/$f1"
    exit $?
else
    echo "font extraction success: $1 -> $out/$f1"
fi
	

	
