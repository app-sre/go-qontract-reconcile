![build](https://ci.ext.devshift.net/buildStatus/icon?job=app-sre-go-qontract-reconcile-gh-build-master)
![license](https://img.shields.io/github/license/app-sre/go-qontract-reconcile.svg?style=flat)


# go-qontract-reconcile

Contains integrations for app-interface for go-qontract-reconcile

### Yaml configuration

```YAML
timeout: Timeout in seconds for the run, defines maximum runtime. (default: 0)
usefeaturetoggle: Weither to check for feature toggles
dryrun: Run in dry run, do not apply resources (default: true)
runonce: Run integration only once (default: false)
sleepdurationsecs: Time to sleep between iterations (default: 600s)

graphql: 
  server: URL to the GraphQL API REQUIRED
  token: Value of Authorization header
  timeout: Timeout for qontract requests (default: 60s) 
  retries: Number of times to retry requests (default: 5)

vault:
  server: Address to access Vault REQUIRED
  authtype: Authentication type either token or approle REQUIRED
  token: Token to access Vault, requires setting authtype to token
  role_id: Role ID to use for authentication, requires setting authtype to approle 
  secret_id: Secret ID to use for authentication, requires setting authtype to approle
  timeout: Timeout for vault requests. (default: 60s) 

user_validator:
  concurrency: Number of coroutines to use to query Github (default: 10)
  invalidusers: Comma seperated list of keys know to be invalid and skipd for PGP key validation

github:
  timeout: Timeout in seconds for Github request (default: 60s)

git_partition_sync_producer:
  glBaseURL: GitLab instance base url
  glUsername: GitLab username associated with token
  glToken: Token to access GitLab. Must have repository read permission
  bucket: s3 bucket to reconcile against
  publicKey: value of x25519 format public key. See https://github.com/FiloSottile/age
  workdir: local dir where git clones and encryption will occur

unleash:
  timeout: Timeout in seconds for Github request (default: 60s)
  apiurl: Address to access Unleash REQUIRED
  clientaccesstoken: Bearer token to use for authentication
```

Configuration can also be passed in as toml, i.e.:

```TOML
[graphql]
server = "https://example/graphql"
token = "Basic Xmjdsfgiohj092w34gjf90erg="

[vault]
server = "https://vault.example.net"
role_id = "a"
secret_id = "b"
```

### Environment variables

Instead of using a yaml file, all parameters can be set via environment variables:
 * DRY_RUN
 * RUN_ONCE
 * RUNNER_TIMEOUT
 * RUNNER_USE_FEATURE_TOGGLE
 * SLEEP_DURATION_SECS
 * GRAPHQL_SERVER
 * GRAPHQL_TIMEOUT
 * GRAPHQL_TOKEN
 * GRAPHQL_RETRIES
 * VAULT_SERVER
 * VAULT_AUTHTYPE
 * VAULT_TOKEN
 * VAULT_ROLE_ID
 * VAULT_SECRET_ID
 * VAULT_TIMEOUT
 * USER_VALIDATOR_CONCURRENCY
 * USER_VALIDATOR_INVALID_USERS
 * UNLEASH_TIMEOUT
 * UNLEASH_API_URL
 * UNLEASH_CLIENT_ACCESS_TOKEN
 * GITHUB_API
 * GITHUB_API_TIMEOUT
 * GITLAB_BASE_URL
 * GITLAB_USERNAME
 * GITLAB_TOKEN
 * PUBLIC_KEY
 * AWS_REGION
 * AWS_GIT_SYNC_BUCKET
 * WORKDIR


## New Integration

If you want to add a new generate you can use the code in `internal/example` as starting point. Copy this folder and give the module a valid go module name. 

Any queries required must be added to the file `generate.go`. Afterwards, update the package parameter in `genqlient.yaml`

Once you updated the graphql files, run the code generator to generate the queries.

`go generate ./...`

This will generate the required code to query `qontract-server`.


## New AWS calls

This code base uses an interface to abstract calls to the AWS SDK. `pkg/awsclient.go`. Benefit of this is, that it enables mocking responses from the AWS SDK. The downside is, that it requires adding used methods to the mentioned interface. After adding the required method, run  `go generate ./...` to generate the corresponding mock code. 


## Authors

These tools have been written by the [Red Hat App-SRE Team](mailto:sd-app-sre@redhat.com).
