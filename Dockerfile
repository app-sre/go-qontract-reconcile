FROM registry.access.redhat.com/ubi9/go-toolset:1.21.13-2.1729776560 as builder
WORKDIR /build
COPY --chown=1001:0 . .
RUN make gobuild

FROM registry.access.redhat.com/ubi9-minimal:9.4
COPY --chown=1001:0 --from=builder /build/go-qontract-reconcile /
RUN microdnf update -y && microdnf install -y ca-certificates git && microdnf clean all
ENTRYPOINT ["/go-qontract-reconcile"]
