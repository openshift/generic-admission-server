#!/bin/bash -e

export GO111MODULE=on
export GOPROXY=https://proxy.golang.org

go mod tidy
