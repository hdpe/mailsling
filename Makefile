all: get-deps generate install

get-deps:
	go get -u github.com/aws/aws-sdk-go \
		github.com/go-sql-driver/mysql \
		github.com/mattes/migrate \
		github.com/a-urth/go-bindata/...

generate:
	(cd ./internal/mailer/schema; go-bindata -pkg schema '.')

install:
	go install hdpe.me/remission-mailer/cmd/mailer
