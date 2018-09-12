#!/bin/sh

# Copyright 2018 The pdfcpu Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#	http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# eg: ./extractPagesDir.sh ~/pdf/big ~/pdf/out

if [ $# -ne 2 ]; then
    echo "usage: ./extractPagesDir.sh inDir outDir"
    echo "generates single-page PDFs for the first 5 pages."
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

    # extract first 5 pages
    pdfcpu extract -verbose -mode=page -pages=-5 $out/$f1/$f $out/$f1 &> $out/$f1/$f1.log
    if [ $? -eq 1 ]; then
        echo "extraction error: $pdf -> $out/$f1"
        echo
		continue
    else
        echo "extraction success: $pdf -> $out/$f1"
        for subpdf in $out/$f1/*_?.pdf
        do
            pdfcpu validate -verbose -mode=relaxed $subpdf >> $out/$f1/$f1.log 2>&1
            if [ $? -eq 1 ]; then
                echo "validation error: $subpdf"
                exit $?
            #else
                #echo "validation success: $subpdf"
            fi
        done
    fi

done
