#!/bin/bash -e

# parse args
deleteFlag=""
while getopts "b:p:d" opt; do
    case $opt in
        b)
            bucket="$OPTARG"
            echo "Using bucket $bucket"
            ;;
        p)
            prefix="$OPTARG"
            prefix=${prefix%"/"}
            prefix=${prefix#"/"}
            echo "Using prefix $prefix"
            ;;
        d)
            deleteFlag="--delete"
            echo "Deleting extra objects"
            ;;
        \?)
            echo "Invalid option: -$OPTARG"
            exit 1
            ;;
    esac
done

# ensure args exist
shouldExit=0
if [ -z "$bucket" ]; then
    echo 'Missing -b (name of bucket to upload code to)'
    shouldExit=1
fi
if [ -z "$prefix" ]; then
    echo 'Missing -p (prefix key of bucket to upload code to)'
    shouldExit=1
fi
if [ $shouldExit -gt 0 ]; then exit 1; fi

# todo: get buckets from stack set (since it's multi-region)

# sync zips
if [ -z "$deleteFlag" ]; then
    aws s3 sync --exclude '*' --include '*.zip' '.' "s3://$bucket/$prefix"
else
    aws s3 sync --exclude '*' --include '*.zip' --delete '.' "s3://$bucket/$prefix"
fi