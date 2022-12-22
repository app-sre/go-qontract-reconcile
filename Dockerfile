FROM quay.io/app-sre/golang:1.18.7 as builder
WORKDIR /build
COPY . .
RUN make gobuild

FROM registry.access.redhat.com/ubi8-minimal
COPY --from=builder /build/go-qontract-reconcile /
RUN microdnf update -y && microdnf install -y ca-certificates && microdnf clean all \
    && chmod 755 /go-qontract-reconcile
ENTRYPOINT ["/go-qontract-reconcile"]
