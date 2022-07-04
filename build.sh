#!/bin/bash

mkdir -p output/bin
go build -o output/bin/monoservice ./cmd/main.go
go build -o output/bin ./test/driver.go
