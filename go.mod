module gitlab.com/gitlab-org/gitlab-workhorse

go 1.12

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/FZambia/sentinel v1.0.0
	github.com/alecthomas/chroma v0.7.3
	github.com/aws/aws-sdk-go v1.31.7
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/getsentry/raven-go v0.1.2
	github.com/golang/gddo v0.0.0-20190419222130-af0f2af80721
	github.com/golang/protobuf v1.4.2
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/gorilla/websocket v1.4.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/jfbus/httprs v0.0.0-20190827093123-b0af8319bb15
	github.com/johannesboyne/gofakes3 v0.0.0-20200510090907-02d71f533bec
	github.com/jpillora/backoff v0.0.0-20170918002102-8eab2debe79d
	github.com/prometheus/client_golang v1.0.0
	github.com/rafaeljusto/redigomock v0.0.0-20190202135759-257e089e14a1
	github.com/sebest/xff v0.0.0-20160910043805-6c115e0ffa35
	github.com/shabbyrobe/gocovmerge v0.0.0-20190829150210-3e036491d500 // indirect
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.5.1
	gitlab.com/gitlab-org/cluster-integration/gitlab-agent v0.0.0-00010101000000-000000000000
	gitlab.com/gitlab-org/gitaly v1.87.1-0.20200519214319-382ead9c7ef3
	gitlab.com/gitlab-org/labkit v0.0.0-20200625061037-a48be4c5e1cc
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b
	golang.org/x/net v0.0.0-20200707034311-ab3426394381
	golang.org/x/tools v0.0.0-20200713011307-fd294ab11aed
	google.golang.org/grpc v1.30.0
	honnef.co/go/tools v0.0.1-2020.1.4
)

replace (
    // TODO Bump dependencies and remove these lines
	gitlab.com/gitlab-org/cluster-integration/gitlab-agent => /Users/mikhail/src/gitlab-agent
	gitlab.com/gitlab-org/labkit => /Users/mikhail/src/labkit
	// https://github.com/kubernetes/kubernetes/issues/79384#issuecomment-505627280
	k8s.io/api => k8s.io/api v0.17.8
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.8
	k8s.io/apiserver => k8s.io/apiserver v0.17.8
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.8
	k8s.io/client-go => k8s.io/client-go v0.17.8
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.8
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.8
	k8s.io/code-generator => k8s.io/code-generator v0.17.8
	k8s.io/component-base => k8s.io/component-base v0.17.8
	k8s.io/cri-api => k8s.io/cri-api v0.17.8
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.8
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.8
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.8
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.8
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.17.8
	k8s.io/kubectl => k8s.io/kubectl v0.17.8
	k8s.io/kubelet => k8s.io/kubelet v0.17.8
	k8s.io/kubernetes => k8s.io/kubernetes v1.17.8 // gitops-engine wants that
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.17.8
	k8s.io/metrics => k8s.io/metrics v0.17.8
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.17.8
)
