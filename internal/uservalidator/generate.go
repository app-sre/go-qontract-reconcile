package uservalidator

//go:generate go run github.com/Khan/genqlient

var _ = `# @genqlient 
query Users  {
    users_v1 {
      path
      name
      org_username
      github_username
      pagerduty_username
      public_gpg_key
    }
  }
  
query GithubOrgs {
    githuborg_v1 {
        name
        token {
        path
        field
        version
        format
        }
        default
    }
}
  
`
