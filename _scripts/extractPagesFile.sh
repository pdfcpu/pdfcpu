#!/bin/sh

#: ./extractPagesFile.sh ~/pdf/1mb/a.pdf ~/pdf/out

if [ $# -ne 2 ]; then
    echo "usage: ./extractPagesFile.sh inFile outDir"
    echo "generates single-page PDFs for the first 5 pages."
    exit 1
fi

f=${1##*/}
f1=${f%.*}
out=$2

#rm -drf $out/*

#set -e

mkdir $out/$f1
cp $1 $out/$f1 

# extract first 5 pages
pdfcpu extract -verbose -mode=page -pages=-5 $out/$f1/$f $out/$f1 &> $out/$f1/$f1.log
if [ $? -eq 1 ]; then
    echo "extraction error: $1 -> $out"
    exit $?
else
    echo "extraction success: $1 -> $out"
    for pdf in $out/$f1/*_?.pdf
    do
        pdfcpu validate -verbose -mode=relaxed $pdf >> $out/$f1/$f1.log 2>&1
        if [ $? -eq 1 ]; then
            echo "validation error: $pdf"
            exit $?
        #else
            #echo "validation success: $pdf"
        fi
    done
fi
	

	
