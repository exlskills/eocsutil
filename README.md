# EOCS Format Utilities

Command-line tool and GoLang API for working with EOCS and OLX courseware. This project also supports importing/updating course to [EXLskills.com](https://exlskills.com/)

## Prerequisites

+ Linux or OS X recommended (PRs for Windows support are welcome)
+ GoLang 1.9.^  (`GOPATH` must be set)
+ GoLang `dep` tool [Install Guide](https://github.com/golang/dep#setup)
+ NodeJS v8.10+
+ Yarn 1.7+

## Installation

```
# Prior to running `go get`, make sure that you have setup Go with the $GOPATH environment variable, otherwise this will not work
go get -u github.com/exlskills/eocsutil
cd $GOPATH/src/github.com/exlskills/eocsutil
dep ensure -v
yarn install
go build # Optional, but this will validate that you have the correct golang deps
```

## Load EOCS course into EXLskills MongoDB

### Assumptions

+ EOCS-formatted course files are placed on the file system  
+ Target MongoDB is running locally (for a production/remote configuration, refer to your sysadmin/internal guides to get the mongodb connection URI; for MongoDB Atlas, use the 3.4 connection URI)

```
export MGO_DB_NAME="<name of the MongoDB target database>"
# Note: `go run` will compile eocsutil on the fly with any code changes, to compile ahead of time, use `go build` and then execute the binary
go run main.go convert --from-format eocs --from-uri <path to the course files folder> --to-format eocs --to-uri mongodb://localhost:27017
```

## FAQ

### I'm getting errors about showdownjs

Sometimes there's an issue with showdownjs nodejs service that eocsutil spawns for markdown<->html conversion not exiting after eocsutil exits. If this occurs, run `ps -ax | grep "node showdownjs/server.js"` and `kill` that process.

### Showdownjs port conflict

Interally, we use a REST API server on http://localhost:6222 to communicate with showdownjs. We're working to select a port on the fly, but until that feature is complete, you will have to either (a) modify eocsutil to use a different port or (b) temporarily stop whatever process is using port 6222 on your machine.

