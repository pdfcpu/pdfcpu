#!/bin/sh

# eg: ./validateDir.sh ~/pdf/big ~/pdf/out

if [ $# -ne 2 ]; then
    echo "usage: ./validateDir.sh inDir outDir"
    exit 1
fi

out=$2

#rm -drf $out/*

#set -e

new=_new

for pdf in $1/*.pdf
do
	#echo $pdf
	
	f=${pdf##*/}
	#echo f = $f
	
	f1=${f%.*}
	#echo f1 = $f1
	
	cp $pdf $out/$f
	
	out1=$out/$f1$new.pdf
	
    pdfcpu validate -verbose -mode=relaxed $out/$f &> $out/$f1.log

    if [ $? -eq 1 ]; then
        echo "validation error: $pdf"
        #exit $?
    else
        echo "validation success: $pdf"
    fi

done
