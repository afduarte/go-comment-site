# Go Comment Site

## Features

* Web socket for instant commenting
* Upvoting
* Giphy integration
* Bolt DB persistence
* Blue Monday Sanitization

## Instructions

```sh
git clone https://github.com/afduarte/go-comment-site.git
cd go-comment-site
go get -u github.com/tidwall/gjson
go get -u github.com/boltdb/bolt
go get -u github.com/microcosm-cc/bluemonday
go get -u github.com/gorilla/websocket
go build src/main.go
./main

```