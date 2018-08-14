# eocsutil
Command-line tool and GoLang API for working with EOCS and OLX courseware

## Prerequisites
+ Linux or OS X recommended  
+ GoLang 1.9.^  (`GOPATH` should be set)  
+ NodeJS v8.10+  
+ Yarn 1.7+  

## Installation
```
go get github.com/exlskills/eocsutil
cd $GOPATH/src/github.com/exlskills/eocsutil
yarn install
```

## Load EOCS course into MongoDB

### Assumptions
+ EOCS - formatted course files are located on the machine's file system  
+ Target MongoDB is running locally

```
export MGO_DB_NAME="<name of the MongoDB target database>"
go run main.go convert --from-format eocs --from-uri <path to the course files folder> --to-format eocs --to-uri mongodb://localhost:27017
```
