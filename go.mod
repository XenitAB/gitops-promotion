module github.com/xenitab/gitops-promotion

go 1.16

require (
	github.com/fluxcd/image-automation-controller v0.6.1
	github.com/fluxcd/image-reflector-controller/api v0.7.0
	github.com/jfrog/jfrog-client-go v0.22.0
	github.com/libgit2/git2go/v31 v31.4.14
	github.com/microsoft/azure-devops-go-api/azuredevops v1.0.0-b5
	github.com/whilp/git-urls v1.0.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/apimachinery v0.20.2
)

replace github.com/fluxcd/image-automation-controller => github.com/fluxcd/image-automation-controller v0.6.2-0.20210303130129-2eebaa46c79b

// side-effect of depending on source-controller
// required by https://github.com/helm/helm/blob/v3.5.2/go.mod
replace (
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
)
