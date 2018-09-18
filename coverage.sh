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

rm c.out

set -e

echo mode: set > c.out

function internalDeps {

    for p in $(go list -f '{{.Deps}}' $1)
    do
        if [[ $p == github.com/hhrutter/pdfcpu* ]]; then
            idep=$idep,$p 
        fi
    done
}

echo collecting coverage ...

for q in $(go list ./...)
do
    #echo collecting coverage for $q
    idep=$q
    internalDeps $idep
    go test -coverprofile=c1.out -coverpkg=$idep $q && tail -n +2 c1.out  >> c.out
done

rm c1.out

go tool cover -html=c.out