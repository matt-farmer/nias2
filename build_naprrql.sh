#!/bin/bash

set -e

CWD=`pwd`


do_build() {
	echo "Building NAPRRQL..."
	mkdir -p $OUTPUT
	cd $CWD
	cd ./app/naprrql
	go get
	GOOS="$GOOS" GOARCH="$GOARCH" go build -ldflags="$LDFLAGS" -o $OUTPUT/$HARNESS
	# cd ..
        # cd $CWD
        # cd ./app/napcomp
        # go get
        # GOOS="$GOOS" GOARCH="$GOARCH" go build -ldflags="$LDFLAGS" -o $OUTPUT/$AUDITDIFFHARNESS
        cd $CWD
	cd ./app
	rsync -a naprrql/naplan_schema.graphql naprrql/public naprrql/school_templates naprrql/system_templates naprrql/in $OUTPUT/
}




do_zip() {
	cd $OUTPUT
	cd ..
	zip -qr ../$ZIP naprrql
	cd $CWD
}

build_mac64() {
	# MAC OS X (64 only)
	echo "Building Mac binaries..."
	GOOS=darwin
	GOARCH=amd64
	LDFLAGS="-s -w"
	OUTPUT=$CWD/build/Mac/naprrql
	# GNATS=nats-streaming-server
	HARNESS=naprrql
        AUDITDIFFHARNESS=napcomp
	ZIP=naprrql-Mac.zip
	do_build
	#do_upx
	# do_shells
	# do_zip
	echo "...all Mac binaries built..."
}


build_windows64() {
	# WINDOWS 64
	echo "Building Windows64 binaries..."
	GOOS=windows
	GOARCH=amd64
	LDFLAGS="-s -w"
	OUTPUT=$CWD/build/Win64/naprrql
	# GNATS=nats-streaming-server.exe
	HARNESS=naprrql.exe
        AUDITDIFFHARNESS=napcomp.exe
	ZIP=naprrql-Win64.zip
	do_build
	#do_upx
	# do_bats
	# do_zip
	echo "...all Windows64 binaries built..."
}

build_windows32() {
	# WINDOWS 32
	echo "Building Windows32 binaries..."
	GOOS=windows
	GOARCH=386
	LDFLAGS="-s -w"
	OUTPUT=$CWD/build/Win32/naprrql
	# GNATS=nats-streaming-server.exe
	HARNESS=naprrql.exe
        AUDITDIFFHARNESS=napcomp.exe
	ZIP=naprrql-Win32.zip
	do_build
	#do_upx
	# do_bats
	# do_zip
	echo "...all Windows32 binaries built..."
}

build_linux64() {
	# LINUX 64
	echo "Building Linux64 binaries..."
	GOOS=linux
	GOARCH=amd64
	LDFLAGS="-s -w"
	OUTPUT=$CWD/build/Linux64/naprrql
	# GNATS=nats-streaming-server
	HARNESS=naprrql
        AUDITDIFFHARNESS=napcomp
	ZIP=naprrql-Linux64.zip
	do_build
	#do_goupx
	# do_shells
	# do_zip
	echo "...all Linux64 binaries built..."
}

build_linux32() {
	# LINUX 32
	echo "Building Linux32 binaries..."
	GOOS=linux
	GOARCH=386
	LDFLAGS="-s -w"
	OUTPUT=$CWD/build/Linux32/naprrql
	# GNATS=nats-streaming-server
	HARNESS=naprrql
        AUDITDIFFHARNESS=napcomp
	ZIP=naprrql-Linux32.zip
	do_build
	#do_goupx
	# do_shells
	# do_zip
	echo "...all Linux32 binaries built..."
}

# TODO ARM
# GOOS=linux GOARCH=arm GOARM=7 go build -o $CWD/build/LinuxArm7/go-nias/aggregator

build_mac64
build_windows64
build_windows32
build_linux64
build_linux32

