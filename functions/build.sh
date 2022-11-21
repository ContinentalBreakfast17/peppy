#!/bin/bash -e

echo "in build"

makefiles=()
while IFS= read -r line; do
    makefiles+=( "$line" )
done < <( find . -name "Makefile" -not -path "./_*-pkg/*" )

for makefile in "${makefiles[@]}"
do
    dir=$(dirname "$makefile")
    echo $dir
    cd $dir && make
    cd ..
done