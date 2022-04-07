# user-validator

User-validator can be used to validate user data stored inside app-interface. 

## usage

`user-validator validate --config config.yaml` 

You can either specify a configuration via _--config_ or set configuration via Environment variables.

### Yaml configuration

```YAML
timeout: Timeout in seconds for the run, defines maximum runtime. (default: 0)

qontract: 
  serverurl: URL to the GraphQL API REQUIRED
  token: Value of Authorization header
  timeout: Timeout for qontract requests (default: 60s) 
  retries: Number of times to retry requests (default: 5)

vault:
  addr: Address to access Vault REQUIRED
  authtype: Authentication type either token or approle REQUIRED
  token: Token to access Vault, requires setting authtype to token
  roleid: Role ID to use for authentication, requires setting authtype to approle 
  secretid: Secret ID to use for authentication, requires setting authtype to approle
  timeout: Timeout for vault requests. (default: 60s) 

user_validator:
  concurrency: Number of coroutines to use to query Github (default: 10)
  invalidusers: Comma seperated list of keys know to be invalid and skipd for pgp key validation

github:
  timeout: Timeout in seconds for Github request (default: 60s)
```

### Environment variables

Instead of using a yaml file, all parameters can be set via environment variables:
 * RUNNER_TIMEOUT
 * QONTRACT_SERVER_URL
 * QONTRACT_TIMEOUT
 * QONTRACT_TOKEN
 * QONTRACT_RETRIES
 * VAULT_ADDR
 * VAULT_AUTHTYPE
 * VAULT_TOKEN
 * VAULT_ROLE_ID
 * VAULT_SECRET_ID
 * VAULT_TIMEOUT
 * USER_VALIDATOR_CONCURRENCY
 * USER_VALIDATOR_INVALID_USERS
 * GITHUB_API_TIMEOUT

## Licence
[Apache License Version 2.0](LICENSE).

## Authors

These tools have been written by the [Red Hat App-SRE Team](mailto:sd-app-sre@redhat.com).
