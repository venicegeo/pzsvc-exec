#!/bin/sh
go test -cover \
  github.com/venicegeo/pzsvc-exec/dispatcher \
  github.com/venicegeo/pzsvc-exec/pzsvc \
  github.com/venicegeo/pzsvc-exec/worker \
  github.com/venicegeo/pzsvc-exec/worker/config \
  github.com/venicegeo/pzsvc-exec/worker/ingest \
  github.com/venicegeo/pzsvc-exec/worker/log \
  github.com/venicegeo/pzsvc-exec/worker/workerexec
