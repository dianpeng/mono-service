#!/bin/bash

find . -type f -name "*.go" -exec go fmt {} \;
