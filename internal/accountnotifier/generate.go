package accountnotifier

//go:generate go run github.com/Khan/genqlient

var _ = `# @genqlient 
query PgpReencryptSettings {
    pgp_reencrypt_settings_v1{
          aws_account_output_vault_path
      reencrypt_vault_path
           private_pgp_key_vault_path
    }
  }
  
query SmtpSettings {
    settings: app_interface_settings_v1 {
        smtp {
        mailAddress
        timeout
        credentials {
            path
            field
            version
            format
        }
        }
    }
}

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
`
