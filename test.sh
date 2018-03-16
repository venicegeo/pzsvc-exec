#!/bin/sh
go test -cover \
  github.com/venicegeo/pzsvc-exec/worker/config \
  github.com/venicegeo/pzsvc-exec/worker/ingest
