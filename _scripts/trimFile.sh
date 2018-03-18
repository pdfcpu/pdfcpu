#!/bin/sh

#: ./trimFile.sh ~/pdf/1mb/a.pdf ~/pdf/1mb/a.

if [ $# -ne 2 ]; then
    echo "usage: ./trimFile.sh inFile outFile"
    echo "generate a PDF with the first 5 pages"
    exit 1
fi

new=_trim

f=${1##*/}
f1=${f%.*}
out=$2

#rm -drf $out/*

#set -e

cp $1 $out/$f 

out1=$out/$f1$new.pdf
pdfcpu trim -verbose -pages=-5 $out/$f $out1 &> $out/$f1.log
if [ $? -eq 1 ]; then
    echo "trim error: $1 -> $out1"
    exit $?
else
    echo "trim success: $1 -> $out1"
    pdfcpu validate -verbose -mode=relaxed $out1 >> $out/$f1.log 2>&1
    if [ $? -eq 1 ]; then
        echo "validation error: $out1"
        exit $?
    else
        echo "validation success: $out1"
    fi    
fi
	

	
