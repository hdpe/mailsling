# stage 1
FROM golang:1.9.4
RUN go get -d -v \
    github.com/aws/aws-sdk-go \
    github.com/go-sql-driver/mysql

COPY cmd/ /go/src/hdpe.me/remission-mailer/cmd/
COPY internal/ /go/src/hdpe.me/remission-mailer/internal/
WORKDIR /go/src/hdpe.me/remission-mailer/cmd/mailer/

RUN CGO_ENABLED=0 GOOS=linux go install -a

# stage 2
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=0 /go/bin/mailer .

CMD ["./mailer"]
