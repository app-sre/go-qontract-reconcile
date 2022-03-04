# user-validator


## usage

`user-validator validate --config config.yaml` 

## Configuration

See the following example for a configuration.yaml. Here authtype is set to `token`. However, a token is not set and might be configurated via `VAULT_TOKEN` environment variable.

```YAML
qontract: 
  server_url: "http://localhost:4000/graphql"

vault:
  addr: "http://localhost:8200"
  authtype: "token"

user_validation:
  concurrent: 10
```