FROM registry.access.redhat.com/ubi9/go-toolset:1.25.3-1764620329@sha256:2e8adc5be8eba060e919a6f7309a7b5ef09dd72d3ae9e747b9eb02e96f35a0f5 as builder
WORKDIR /build
COPY --chown=1001:0 . .

ENV PATH="$HOME/go/bin:$PATH"
RUN go install github.com/golang/mock/mockgen && \
    go install github.com/Khan/genqlient

# Linting, build and unit tests
RUN make generate golint gobuild

FROM registry.access.redhat.com/ubi9-minimal:9.7-1764794109@sha256:6fc28bcb6776e387d7a35a2056d9d2b985dc4e26031e98a2bd35a7137cd6fd71 as prod
COPY --chown=1001:0 --from=builder /build/go-qontract-reconcile /
COPY --chown=1001:0 --from=builder /build/licenses/LICENSE /licenses/LICENSE
RUN microdnf update -y && microdnf install -y ca-certificates git && microdnf clean all
USER 1001
ENTRYPOINT ["/go-qontract-reconcile"]
