#!/bin/sh

# eg: ./optimizeDir.sh ~/pdf/big ~/pdf/out

if [ $# -ne 2 ]; then
    echo "usage: ./optimizeDir.sh inDir outDir"
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
	pdfcpu optimize -verbose -stats=stats.csv $out/$f $out1 &> $out/$f1.log
	if [ $? -eq 1 ]; then
        echo "optimization error: $pdf -> $out1"
        echo
		continue
    else
        echo "optimization success: $pdf -> $out1"
    fi
	
	out2=$out/$f1$new$new.pdf
	pdfcpu optimize -verbose -stats=statsNew.csv $out1 $out2 &> $out/$f1$new.log
	if [ $? -eq 1 ]; then
        echo "optimization error: $out1 -> $out2"
    else
        echo "optimization success: $out1 -> $out2"
    fi
	
	echo
	
done
