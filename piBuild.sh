#!/bin/bash

cd codeGen
go run .
if [ $? -ne 0 ]; then
  echo "Code Generation Failed"
  exit 1
fi
echo "Code Generation OK"
cd ..

go build .
if [ $? -ne 0 ]; then
  echo "Build Failed"
  exit 1
fi

echo "Build OK..."
