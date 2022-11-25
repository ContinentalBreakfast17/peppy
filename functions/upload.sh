#!/bin/bash -e

# parse args
deleteFlag=""
while getopts "s:p:d" opt; do
    case $opt in
        s)
            stack="$OPTARG"
            echo "Using stack $stack"
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
if [ -z "$stack" ]; then
    echo 'Missing -s (name of stack containing relevant artifact parameters)'
    shouldExit=1
fi
if [ -z "$prefix" ]; then
    echo 'Missing -p (object prefix key to upload code to)'
    shouldExit=1
fi
if [ $shouldExit -gt 0 ]; then exit 1; fi

# get buckets/regions from stack set
params=$(aws cloudformation describe-stacks --stack-name "$stack" | jq -r '.Stacks[0].Parameters')
regions=()
bucket_prefix=""
for row in $(echo "${params}" | jq -r '.[] | @base64'); do
    _jq() {
        echo ${row} | base64 --decode | jq -r ${1}
    }

    if [ "$(_jq '.ParameterKey')" = "StackSetBucketPrefix" ]; then
        bucket_prefix="$(_jq '.ParameterValue')"
    elif [ "$(_jq '.ParameterKey')" = "Regions" ]; then
        IFS=',' read -ra regions <<< "$(_jq '.ParameterValue')"
    fi
done

# sync zips
for region in "${regions[@]}"; do
    bucket="${bucket_prefix}-${region}"
    echo "Uploading to $bucket"

    if [ -z "$deleteFlag" ]; then
        aws s3 sync --exclude '*' --include '*.zip' '.' "s3://$bucket/$prefix"
    else
        aws s3 sync --exclude '*' --include '*.zip' --delete '.' "s3://$bucket/$prefix"
    fi
done