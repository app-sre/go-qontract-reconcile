FROM registry.access.redhat.com/ubi9/go-toolset:1.23.9-1750969886@sha256:3bbd87d77ea93742bd71a5275a31ec4a7693454ab80492c6a7d28ce6eef35378 as builder
WORKDIR /build
COPY --chown=1001:0 . .

ENV PATH="$HOME/go/bin:$PATH"
RUN go install github.com/golang/mock/mockgen && \
    go install github.com/Khan/genqlient

# Linting, build and unit tests
RUN make generate golint gobuild

FROM registry.access.redhat.com/ubi9-minimal:9.6-1750782676@sha256:e12131db2e2b6572613589a94b7f615d4ac89d94f859dad05908aeb478fb090f as prod
COPY --chown=1001:0 --from=builder /build/go-qontract-reconcile /
COPY --chown=1001:0 --from=builder /build/licenses/LICENSE /licenses/LICENSE
RUN microdnf update -y && microdnf install -y ca-certificates git && microdnf clean all
USER 1001
ENTRYPOINT ["/go-qontract-reconcile"]
