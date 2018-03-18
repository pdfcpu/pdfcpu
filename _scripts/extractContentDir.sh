#!/bin/sh

# eg: ./extractContentDir.sh ~/pdf/big ~/pdf/out

if [ $# -ne 2 ]; then
    echo "usage: ./extractContentDir.sh inDir outDir"
    exit 1
fi

out=$2

#rm -drf $out/*

#set -e

for pdf in $1/*.pdf
do
	
	f=${pdf##*/}
	#echo f = $f
	
	f1=${f%.*}
	#echo f1 = $f1
	
    mkdir $out/$f1
    cp $pdf $out/$f1

    pdfcpu extract -verbose -mode=content $out/$f1/$f $out/$f1 &> $out/$f1/$f1.log
    if [ $? -eq 1 ]; then
        echo "extraction error: $pdf -> $out/$f1"
        echo
		continue
    else
        echo "extraction success: $pdf -> $out/$f1"
    fi

done
