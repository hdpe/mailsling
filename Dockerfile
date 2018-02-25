# stage 1
FROM golang:1.9.4

COPY Makefile /go/src/hdpe.me/remission-mailer/
COPY cmd/ /go/src/hdpe.me/remission-mailer/cmd/
COPY internal/ /go/src/hdpe.me/remission-mailer/internal/

WORKDIR /go/src/hdpe.me/remission-mailer/
RUN CGO_ENABLED=0 GOOS=linux make

# stage 2
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=0 /go/bin/mailer .

RUN echo '* * * * * /root/mailer' >> /var/spool/cron/crontabs/root

CMD ["crond", "-f"]
