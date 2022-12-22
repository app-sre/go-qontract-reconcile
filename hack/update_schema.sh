#!/bin/bash 
set -e

container_engine=$(shell which podman >/dev/null 2>&1 && echo podman || echo docker)
git_commit=$(git rev-parse HEAD)
git_commit_timestamp=$(git log -1 --format=%ct ${git_commit})
output_dir=$(echo $(realpath pwd)/bundle)
bundle=data.json
image=quay.io/app-sre/go-qontract-reconcile
qontract_server_image=quay.io/app-sre/qontract-server
qontract_schemas_image=quay.io/app-sre/qontract-schemas

if [[ -d ${PWD}/.docker ]]
then
	docker_conf=${PWD}/.docker
else
	docker_conf=${HOME}/.docker
fi

mkdir -p ${output_dir} ${PWD}/fake_data $PWD/fake_resources

function cleanup {
    rm -rf ${PWD}/fake_data ${PWD}/fake_resources ${PWD}/schemas ${PWD}/graphql-schemas
    ${container_engine} stop goqr-qr-server
    ${container_engine} network rm goqr-shared || true
}
trap cleanup EXIT

${container_engine} create --name=goqr-qr-schemas ${qontract_schemas_image}:latest
${container_engine} cp goqr-qr-schemas:/schemas/. .
${container_engine} rm goqr-qr-schemas &>/dev/null

${container_engine} run --rm \
    -v ${PWD}/schemas:/schemas:z \
    -v ${PWD}/graphql-schemas:/graphql:z \
    -v ${PWD}/fake_data:/data:z \
    -v ${PWD}/fake_resources:/resources:z \
    ${image}:latest \
    qontract-bundler /schemas /graphql/schema.yml /data /resources ${git_commit} ${git_commit_timestamp} > ${output_dir}/${bundle}


[[ $(${container_engine} network ls | grep -w goqr-shared -c) -eq 0  ]] && ${container_engine} network create goqr-shared

${container_engine} run -it --rm \
    -v ${output_dir}:/bundle:z \
    -p 4000:4000 -d \
    --name=goqr-qr-server \
    -e LOAD_METHOD=fs --network=goqr-shared \
    -e DATAFILES_FILE=/bundle/${bundle} \
    ${qontract_server_image}:latest


while true
do
    [[ $(curl -sw '%{http_code}' http://localhost:4000/healthz) -eq 200 ]] && break
    echo "waiting for server..."
    sleep 0.1
done

if [[ ${container_engine} == "docker" ]]
then
    ${container_engine} --config=${docker_conf} build --no-cache -t gql:latest -f ./hack/Dockerfile.gql .
else
    ${container_engine} build --no-cache -t gql:latest -f ./hack/Dockerfile.gql .
fi

${container_engine} run --rm --network=goqr-shared gql gql-cli \
    http://goqr-qr-server:4000/graphql --print-schema  > schema.graphql

