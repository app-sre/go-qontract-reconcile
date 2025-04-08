FROM registry.access.redhat.com/ubi9/go-toolset:1.22.9-1743582279@sha256:42c9557a27ecb3909796ad47170ad6c06a023fde89588526cb8f2b0e4e6bae84 as builder
WORKDIR /build
COPY --chown=1001:0 . .

ENV PATH="$HOME/go/bin:$PATH"
RUN go install github.com/golang/mock/mockgen && \
    go install github.com/Khan/genqlient

# Linting, build and unit tests
RUN make generate golint gobuild

FROM registry.access.redhat.com/ubi9-minimal:9.5-1742914212@sha256:ac61c96b93894b9169221e87718733354dd3765dd4a62b275893c7ff0d876869 as prod
COPY --chown=1001:0 --from=builder /build/go-qontract-reconcile /
COPY --chown=1001:0 --from=builder /build/licenses/LICENSE /licenses/LICENSE
RUN microdnf update -y && microdnf install -y ca-certificates git && microdnf clean all
USER 1001
ENTRYPOINT ["/go-qontract-reconcile"]
