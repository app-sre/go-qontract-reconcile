FROM quay.io/app-sre/golang:1.17 as builder
WORKDIR /build
COPY . .
RUN make gobuild

FROM registry.access.redhat.com/ubi8-minimal
COPY --from=builder /build/user-validator /
RUN microdnf update -y && microdnf install -y ca-certificates && microdnf clean all \
    && chmod 755 /user-validator
ENTRYPOINT ["/user-validator"]
