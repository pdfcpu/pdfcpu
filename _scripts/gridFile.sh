#!/bin/sh

# Copyright 2019 The pdfcpu Authors.
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

# eg: ./gridFile.sh ~/pdf/1mb/a.pdf ~/pdf/out

if [ $# -ne 2 ]; then
    echo "usage: ./gridFile.sh inFile outDir"
    echo "rearrange all pages into 1x3 page grids"
    exit 1
fi

new=_grid

f=${1##*/}
f1=${f%.*}
out=$2

cp $1 $out/$f 

out1=$out/$f1$new.pdf
pdfcpu grid -verbose $out1 1 3 $out/$f &> $out/$f1.log
if [ $? -eq 1 ]; then
    echo "grid error: $1 -> $out1"
    exit $?
else
    echo "grid success: $1 -> $out1"
    pdfcpu validate -verbose -mode=relaxed $out1 >> $out/$f1.log 2>&1
    if [ $? -eq 1 ]; then
        echo "validation error: $out1"
        exit $?
    else
        echo "validation success: $out1"
    fi    
fi
	

	
