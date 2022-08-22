#!/bin/sh

protoc -I=. --gofast_out=paths=source_relative,plugins=grpc:. storage.proto
