all: get-deps generate install test

get-deps:
	go get -u github.com/aws/aws-sdk-go \
		github.com/go-sql-driver/mysql \
		github.com/mattes/migrate \
		github.com/a-urth/go-bindata/...

generate:
	(cd ./internal/mailer/schema; go-bindata -pkg schema '.')

test:
	go test ./...

install:
	go install github.com/hdpe/mailsling/cmd/mailsling
