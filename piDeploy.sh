#!/bin/bash

go clean
rm -rf tempdeploy
mkdir tempdeploy
cp *.go tempdeploy/
cp go.mod tempdeploy/
cp configThumbnailPi.json tempdeploy/
cp configThumbnailPiFull.json tempdeploy/
cp piBuild.sh tempdeploy/
cp piRun.sh tempdeploy/
rsync -avz -e 'ssh' tempdeploy/ pi@192.168.1.243:/home/pi/server/thumbnailGen/
rm -rf tempdeploy