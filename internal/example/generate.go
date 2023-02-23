package example

//go:generate go run github.com/Khan/genqlient

var _ = `# @genqlient 
query Users  {
    users_v1 {
        path
        name
        org_username
        github_username
        slack_username
        pagerduty_username
        public_gpg_key
    }
}
`
