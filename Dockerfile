FROM quay.io/app-sre/golang:1.22.7 as builder
WORKDIR /build
COPY . .
RUN make gobuild

FROM registry.access.redhat.com/ubi9-minimal:9.4
COPY --from=builder /build/go-qontract-reconcile /
RUN microdnf update -y && microdnf install -y ca-certificates && microdnf clean all \
    && microdnf install -y git \
    && chmod 755 /go-qontract-reconcile
ENTRYPOINT ["/go-qontract-reconcile"]
