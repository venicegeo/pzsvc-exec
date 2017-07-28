#!/bin/bash

# Helper shell script to upload the conda repo to nexus
for f in $(find linux-64 -type f); do
  echo Sending "$f" to https://nexus.devops.geointservices.io/content/repositories/beachfront-conda/$f
  curl -T "$f" -u $NEXUSUSER:$NEXUSPASS https://nexus.devops.geointservices.io/content/repositories/beachfront-conda/$f
done
