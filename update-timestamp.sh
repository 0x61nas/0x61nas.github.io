#!/usr/bin/env bash

#!/bin/bash

# Define the file containing the meta tag
file="index.html"

# Get the current timestamp in the required format (e.g., 2024-10-09T15:45:00-07:00)
current_timestamp=$(date +"%Y-%m-%dT%H:%M:%S%:z")

# Use sed to update the content of the og:updated_time meta tag
sed -i "s|<meta property=\"og:updated_time\" content=\"[^\"]*\" />|<meta property=\"og:updated_time\" content=\"$current_timestamp\" />|" "$file"

echo "og:updated_time has been updated to $current_timestamp"

