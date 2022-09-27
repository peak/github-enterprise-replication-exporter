FROM golang:1.17.0-alpine AS build

ENV GO111MODULE=on

WORKDIR /app

COPY . ./

RUN apk --no-cache add git ca-certificates
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X 'main.version=`git describe --tags --abbrev=0`'" .

FROM alpine:3.14.2

RUN apk --no-cache add git ca-certificates
COPY --from=build /app/github-enterprise-replication-exporter /github-enterprise-replication-exporter
ENTRYPOINT [ "/github-enterprise-replication-exporter" ]
CMD [ "" ]