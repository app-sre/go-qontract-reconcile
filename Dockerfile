FROM quay.io/app-sre/golang:1.18.7 as builder
WORKDIR /build
COPY . .
RUN make gobuild

FROM registry.access.redhat.com/ubi8-minimal:8.8
COPY --from=builder /build/go-qontract-reconcile /
RUN microdnf update -y && microdnf install -y ca-certificates && microdnf clean all \
    && microdnf install -y git \
    && chmod 755 /go-qontract-reconcile
ENTRYPOINT ["/go-qontract-reconcile"]
