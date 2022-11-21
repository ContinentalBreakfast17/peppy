#!/bin/bash -e

echo "in build"
pwd
ls -la

makefiles=()
while IFS= read -r line; do
    makefiles+=( "$line" )
done < <( find . -name "*.Makefile" )

for makefile in "${makefiles[@]}"
do
    dir=$(dirname "$makefile")
    echo $dir
    cd $dir && make
    cd ..
done