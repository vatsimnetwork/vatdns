FROM golang:1.19-alpine AS build


COPY . /go/src/vatdns
WORKDIR /go/src/vatdns

RUN apk add git
RUN go build -o /bin/dnshaiku cmd/dnshaiku/main.go
RUN go build -o /bin/retardantfoam cmd/retardantfoam/main.go

FROM scratch
COPY --from=build /bin/dnshaiku /bin/dnshaiku
COPY --from=build /bin/retardantfoam /bin/retardantfoam
