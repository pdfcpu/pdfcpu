#!/bin/sh

#: ./optimizeFile.sh ~/pdf/1mb/a.pdf ~/pdf/out

if [ $# -ne 2 ]; then
    echo "usage: ./optimizeFile.sh inFile outDir"
    exit 1
fi

new=_new

f=${1##*/}
f1=${f%.*}
out=$2

#rm -drf $out/*

#set -e

cp $1 $out/$f

out1=$out/$f1$new.pdf
pdfcpu optimize -verbose $out/$f $out1 &> $out/$f1.log
if [ $? -eq 1 ]; then
    echo "optimization error: $1 -> $out1"
    exit $?
else
    echo "optimization success: $1 -> $out1"
fi
	
out2=$out/$f1$new$new.pdf
pdfcpu optimize -verbose $out1 $out2 &> $out/$f1$new.log
if [ $? -eq 1 ]; then
    echo "optimization error: $out1 -> $out2"
    exit $?
else
    echo "optimization success: $out1 -> $out2"
fi

	
