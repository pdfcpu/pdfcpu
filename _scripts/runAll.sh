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

if [ $# -lt 2 ]; then
    echo "usage: ./runAll.sh outDir dir..."
    exit 1
fi

out=$1

rm -drf $out/*
for dir in $*; do
    if [ $dir = $1 ]; then
        continue
    fi
    echo $dir
    ./validateDir.sh $dir $out
done

rm -drf $out/*
for dir in $*; do
    if [ $dir = $1 ]; then
        continue
    fi
    echo $dir
    ./optimizeDir.sh $dir $out
done