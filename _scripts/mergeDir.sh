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

# eg: ./mergeDir.sh ~/pdf/big ~/pdf/out

if [ $# -ne 2 ]; then
    echo "usage: ./mergeDir.sh inDir outDir"
    exit 1
fi

out=$2

#rm -drf $out/*

#set -e

for pdf in $1/*.pdf
do
	f=${pdf##*/}
	cp $pdf $out/$f
done

pdfcpu merge -verbose $out/merged.pdf $out/*.pdf &> $out/merged.log
if [ $? -eq 1 ]; then
	echo "merge error: $1/*.pdf -> $out/merged.pdf"
    echo
	continue
else
    echo "merge success: $1/*.pdf -> $out/merged.pdf"
fi