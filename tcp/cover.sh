#!/bin/bash
go test -coverprofile=testdata/stats.out  && go tool cover -html=testdata/stats.out
