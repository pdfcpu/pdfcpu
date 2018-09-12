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
