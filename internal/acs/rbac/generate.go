// Package example contains an example integration
package rbac

//go:generate go run github.com/Khan/genqlient

var _ = `# @genqlient 
query GetAcsOidcPermissions  {
    oidc_permissions_v1: oidc_permissions_v1 {
        name
        description
        service
        ... on OidcPermissionAcs_v1 {
            permission_set
            clusters {
                name
            }
            namespaces {
                name
                cluster {
                    name
                }
            }
        }
    }
}
`
