FROM golang:1.24

WORKDIR /build
COPY . ./
RUN CGO_ENABLED=0 go build -a -tags netgo -ldflags '-w' -o rtbackup ./

FROM docker:28.1.1-dind-alpine3.21

RUN apk add --no-cache ca-certificates tzdata curl restic
COPY --from=0 /build/rtbackup /bin/
ENTRYPOINT [ "/bin/rtbackup" ]