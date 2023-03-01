# Git Partition Sync - Producer
Uploads encrypted/zipped latest versions of target GitLab projects to s3 bucket.  
This works in tandem with [git-partition-sync-consumer](https://github.com/app-sre/git-partition-sync-consumer) to sync GitLab instances in isolated environments.

[age](https://github.com/FiloSottile/age) x25519 format keys are utilized.

![gitlab-sync-diagram](../gitsync-diagram.png)

## Uploaded s3 Object Key Format
Uploaded keys are base64 encoded. Decoded, the key is a json string with following structure:
```
{
  "group":string,
  "project_name":string,
  "commit_sha":string,
  "local_branch":string,
  "remote_branch":string
}
```
**Note:** the values within each json will mirror values for each `destination` defined within config file (exluding `commit_sha` which is the latest commit pulled from `source`)
