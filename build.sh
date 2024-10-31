#!/bin/env bash

platforms=("linux/amd64" "windows/amd64" "darwin/amd64" "darwin/arm64")

for platform in "${platforms[@]}"
do
	platform_split=(${platform//\// })
	GOOS=${platform_split[0]}
	GOARCH=${platform_split[1]}
	
    output_name="dist/endo-rivens-$GOOS-$GOARCH"

	if [ $GOOS = "windows" ]; then
		output_name+=".exe"
	fi

	env GOOS=$GOOS GOARCH=$GOARCH go build -o $output_name

	if [ $? -ne 0 ]; then
   		echo "Failed to build for $platform"
		exit 1
	else
		echo "Built $output_name"
	fi
done
