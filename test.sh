#!/bin/sh
go test -cover \
  github.com/venicegeo/pzsvc-exec/dispatcher \
  github.com/venicegeo/pzsvc-exec/dispatcher/cfwrapper \
  github.com/venicegeo/pzsvc-exec/dispatcher/model \
  github.com/venicegeo/pzsvc-exec/dispatcher/poll \
  github.com/venicegeo/pzsvc-exec/pzsvc \
  github.com/venicegeo/pzsvc-exec/worker \
  github.com/venicegeo/pzsvc-exec/worker/config \
  github.com/venicegeo/pzsvc-exec/worker/ingest \
  github.com/venicegeo/pzsvc-exec/worker/input \
  github.com/venicegeo/pzsvc-exec/worker/log \
  github.com/venicegeo/pzsvc-exec/worker/workerexec
