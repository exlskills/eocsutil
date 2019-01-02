# EOCS Format Utilities

Command-line tool and GoLang API for working with EOCS and OLX courseware. This project also supports importing/updating course to [EXLskills.com](https://exlskills.com/)

## Prerequisites

+ Linux or OS X recommended (PRs for Windows support are welcome)
+ GoLang 1.10.^  (`GOPATH` must be set)
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

## Load EOCS course into EXLskills MongoDB and Elasticsearch (version 6.x)

### Assumptions

+ EOCS-formatted course files are placed on the file system  
+ Target MongoDB is running locally (for a production/remote configuration, refer to your sysadmin/internal guides to get the mongodb connection URI; for MongoDB Atlas, use the 3.4 connection URI)

```
export MGO_DB_NAME="<name of the MongoDB target database>"

# Note: if Elasticsearch URI is not provided - the indexing will be bypassed
export ELASTICSEARCH_URI="http://localhost:9200"

# The default value for Elasticsearch "base" index is "learn". The actual index name will be set to the base plus "_<course launguage>", e.g., "base_en"
# To override the base name, set the Environment variable as below 
export ELASTICSEARCH_BASE_INDEX="learn"
 
# Note: `go run` will compile eocsutil on the fly with any code changes, to compile ahead of time, use `go build` and then execute the binary
# MongoDB URI *must* start with `mongodb:` - version 3.4 style
go run main.go convert --from-format eocs --from-uri <path to the course files folder> --to-format eocs --to-uri mongodb://localhost:27017
```

### Connecting to Elasticsearch HTTPS Backend in a non-production mode 

Some Elasticsearch security models, e.g., AWS VPC Elasticsearch Service, require HTTPS connectivity in either Production or Test modes. To bypass Certificate validation in testing, ensure to set
```
export MODE=debug
```

A full example of connecting to AWS VPC Elasticsearch Service for testing:
```
// This is run in a separate terminal session that creates a tunnel to the AWS VPC via any instance running in the VPC:
ssh -i ~/.ssh/myAwsInstanceKey ec2-user@123.4.5.6 -N -L 19200:vpc-my-es-service.us-west-2.es.amazonaws.com:443

// This is run in the eocsutil testing terminal session: 
export MODE=debug
export ELASTICSEARCH_URI="https://localhost:19200"
```
  

## FAQ

### I'm getting errors about showdownjs

Sometimes there's an issue with showdownjs nodejs service that eocsutil spawns for markdown<->html conversion not exiting after eocsutil exits. If this occurs, run `ps -ax | grep "node --harmony showdownjs/server.js"` and `kill` that process.

### Showdownjs port conflict

Internally, we use a REST API server on http://localhost:6222 to communicate with showdownjs. We're working to select a port on the fly, but until that feature is complete, you will have to either (a) modify eocsutil to use a different port or (b) temporarily stop whatever process is using port 6222 on your machine.

