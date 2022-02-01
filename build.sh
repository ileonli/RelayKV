#!/bin/bash

echo "rm older server binary"
rm ./bin/server

echo "build server binary"
go build -race -o ./bin ./server/

echo "rm older image"
docker rmi relaykv

echo "build new image"
docker build -t relaykv .