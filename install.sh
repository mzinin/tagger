#! /bin/sh

major=`cat ./version.json | grep -m 1 Major | grep -o '[0-9]*'`
minor=`cat ./version.json | grep -m 1 Minor | grep -o '[0-9]*'`
patch=`cat ./version.json | grep -m 1 Patch | grep -o '[0-9]*'`

go install -ldflags "-X main.version=$major.$minor.$patch"
