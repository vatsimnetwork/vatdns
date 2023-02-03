FROM golang:1.19-alpine AS build


COPY . /usr/local/go/src/vatdns
WORKDIR /usr/local/go/src/vatdns

RUN apk add git
RUN go build -o /bin/dataprocessor cmd/dataprocessor/main.go
RUN go build -o /bin/dnshaiku cmd/dnshaiku/main.go
RUN go build -o /bin/retardantfoam cmd/retardantfoam/main.go

FROM scratch
COPY --from=build /bin/dataprocessor /bin/dataprocessor
COPY --from=build /bin/dnshaiku /bin/dnshaiku
COPY --from=build /bin/retardantfoam /bin/retardantfoam
