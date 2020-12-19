module github.com/ryota-sakamoto/cert-manager-external-sample

go 1.13

require (
	github.com/go-logr/logr v0.2.1-0.20200730175230-ee2de8da5be6
	github.com/go-logr/zapr v0.3.0 // indirect
	github.com/jetstack/cert-manager v1.1.0
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v0.19.0
	sigs.k8s.io/controller-runtime v0.6.2
)
