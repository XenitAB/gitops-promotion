module github.com/xenitab/gitops-promotion

go 1.16

require (
	github.com/andybalholm/brotli v1.0.2 // indirect
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/fluxcd/image-automation-controller v0.12.0
	github.com/fluxcd/image-reflector-controller/api v0.10.0
	github.com/fluxcd/source-controller v0.15.2
	github.com/go-errors/errors v1.1.1 // indirect
	github.com/go-openapi/analysis v0.20.1 // indirect
	github.com/go-openapi/errors v0.20.0 // indirect
	github.com/go-openapi/runtime v0.19.28 // indirect
	github.com/go-openapi/strfmt v0.20.1 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/go-openapi/validate v0.20.2 // indirect
	github.com/google/go-containerregistry v0.5.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/jfrog/jfrog-client-go v0.24.0
	github.com/klauspost/compress v1.12.2 // indirect
	github.com/klauspost/pgzip v1.2.5 // indirect
	github.com/libgit2/git2go/v31 v31.4.14
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/microsoft/azure-devops-go-api/azuredevops v1.0.0-b5
	github.com/pierrec/lz4/v4 v4.1.6 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/ulikunitz/xz v0.5.10 // indirect
	github.com/whilp/git-urls v1.0.0
	github.com/xlab/treeprint v1.1.0 // indirect
	golang.org/x/term v0.0.0-20210429154555-c04ba851c2a4 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/apimachinery v0.21.2
	sigs.k8s.io/kustomize/kyaml v0.10.17 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.1.1 // indirect
)

replace github.com/fluxcd/image-automation-controller => github.com/fluxcd/image-automation-controller v0.6.2-0.20210303130129-2eebaa46c79b

// side-effect of depending on source-controller
// required by https://github.com/helm/helm/blob/v3.5.2/go.mod
replace (
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
)
