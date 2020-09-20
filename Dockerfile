FROM golang:1.14 AS builder
ENV CGO_ENABLED 0
WORKDIR /go/src/app
ADD . .
RUN go build -mod vendor -o /auto-fix-tke-ingress

FROM alpine:3.12
COPY --from=builder /auto-fix-tke-ingress /auto-fix-tke-ingress
CMD ["/auto-fix-tke-ingress"]