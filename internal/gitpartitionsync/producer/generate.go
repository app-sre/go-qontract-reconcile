package producer

//go:generate go run github.com/Khan/genqlient

var _ = `# @genqlient 
query GetSaasResourceTemplateRefs {
    saas_files: saas_files_v2 {
        name
        resourceTemplates {
            targets {
                ref
            }
        }
    },
}
`
var _ = `# @genqlient 
query GetGitlabSyncApps {
    apps_v1: apps_v1 {
        codeComponents {
            gitlabSync {
                ... on CodeComponentGitlabSync_v1 {
                    sourceProject {
                        ... on CodeComponentGitlabSyncProject_v1 {
                            name
                            group
                            branch
                        }
                    }
                    destinationProject {
                        ... on CodeComponentGitlabSyncProject_v1 {
                            name
                            group
                            branch
                        }
                    }
                }
            }
        }
    }
}
`
