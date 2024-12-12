FROM registry.access.redhat.com/ubi9/go-toolset:1.21.13-2.1729776560@sha256:97e30a01caeece72ee967013e7c7af777ea4ee93840681ddcfe38a87eb4c084a as builder
WORKDIR /build
COPY --chown=1001:0 . .

ENV PATH="$HOME/go/bin:$PATH"
RUN go install github.com/golang/mock/mockgen && \
    go install github.com/Khan/genqlient

# Linting, build and unit tests
RUN make generate golint gobuild

FROM registry.access.redhat.com/ubi9-minimal:9.5-1733767867@sha256:dee813b83663d420eb108983a1c94c614ff5d3fcb5159a7bd0324f0edbe7fca1 as prod
COPY --chown=1001:0 --from=builder /build/go-qontract-reconcile /
COPY --chown=1001:0 --from=builder /build/licenses/LICENSE /licenses/LICENSE
RUN microdnf update -y && microdnf install -y ca-certificates git && microdnf clean all
USER 1001
ENTRYPOINT ["/go-qontract-reconcile"]
