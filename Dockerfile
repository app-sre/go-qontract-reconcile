FROM registry.access.redhat.com/ubi9/go-toolset:1.24.4-1755074415@sha256:81d9cb51feb0b56e3d87a73c2237bebaa80b28e4d5791c1da1140eae50225e2f as builder
WORKDIR /build
COPY --chown=1001:0 . .

ENV PATH="$HOME/go/bin:$PATH"
RUN go install github.com/golang/mock/mockgen && \
    go install github.com/Khan/genqlient

# Linting, build and unit tests
RUN make generate golint gobuild

FROM registry.access.redhat.com/ubi9-minimal:9.6-1754584681@sha256:8d905a93f1392d4a8f7fb906bd49bf540290674b28d82de3536bb4d0898bf9d7 as prod
COPY --chown=1001:0 --from=builder /build/go-qontract-reconcile /
COPY --chown=1001:0 --from=builder /build/licenses/LICENSE /licenses/LICENSE
RUN microdnf update -y && microdnf install -y ca-certificates git && microdnf clean all
USER 1001
ENTRYPOINT ["/go-qontract-reconcile"]
