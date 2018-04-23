#!/usr/bin/env bash


psql -l
while [ $? -ne 0 ]; do psql -l && sleep 1; done # wait for postgres warm up
createdb metrics_iterator || true

psql -U root -d metrics_iterator -a -f /code/migration.sql