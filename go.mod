module github.com/cyberark/sidecar-injector

go 1.19

require (
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/cyberark/conjur-api-go v0.10.2
	github.com/cyberark/conjur-authn-k8s-client v0.23.8
	github.com/cyberark/conjur-opentelemetry-tracer v1.55.55
	github.com/cyberark/secrets-provider-for-k8s v1.4.4
	github.com/evanphx/json-patch v5.6.0+incompatible
	github.com/stretchr/testify v1.8.1
	go.opentelemetry.io/otel v1.11.1
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/api v0.25.4
	k8s.io/apimachinery v0.25.4
	k8s.io/client-go v0.25.4
)

require (
	github.com/PuerkitoBio/purell v1.2.0 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/bgentry/go-netrc v0.0.0-20140422174119-9fd32a8b3d3d // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/go-restful/v3 v3.10.0 // indirect
	github.com/fullsailor/pkcs7 v0.0.0-20190404230743-d7302db945fa // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/gnostic v0.6.9 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	go.opentelemetry.io/otel/exporters/jaeger v1.11.1 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.11.1 // indirect
	go.opentelemetry.io/otel/sdk v1.11.1 // indirect
	go.opentelemetry.io/otel/trace v1.11.1 // indirect
	golang.org/x/net v0.2.0 // indirect
	golang.org/x/oauth2 v0.2.0 // indirect
	golang.org/x/sys v0.2.0 // indirect
	golang.org/x/term v0.2.0 // indirect
	golang.org/x/text v0.4.0 // indirect
	golang.org/x/time v0.2.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/klog/v2 v2.80.1 // indirect
	k8s.io/kube-openapi v0.0.0-20221110221610-a28e98eb7c70 // indirect
	k8s.io/utils v0.0.0-20221108210102-8e77b1f39fe2 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace github.com/cyberark/conjur-opentelemetry-tracer v1.55.55 => github.com/cyberark/conjur-opentelemetry-tracer v0.0.1-655

replace golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2 => golang.org/x/crypto v0.0.0-20220314234659-1baeb1ce4c0b

replace golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550 => golang.org/x/crypto v0.0.0-20220314234659-1baeb1ce4c0b

replace golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9 => golang.org/x/crypto v0.0.0-20220314234659-1baeb1ce4c0b

replace golang.org/x/crypto v0.0.0-20210921155107-089bfa567519 => golang.org/x/crypto v0.0.0-20220314234659-1baeb1ce4c0b

replace golang.org/x/net v0.0.0-20180724234803-3673e40ba225 => golang.org/x/net v0.0.0-20220923203811-8be639271d50

replace golang.org/x/net v0.0.0-20180826012351-8a410e7b638d => golang.org/x/net v0.0.0-20220923203811-8be639271d50

replace golang.org/x/net v0.0.0-20180906233101-161cd47e91fd => golang.org/x/net v0.0.0-20220923203811-8be639271d50

replace golang.org/x/net v0.0.0-20190213061140-3a22650c66bd => golang.org/x/net v0.0.0-20220923203811-8be639271d50

replace golang.org/x/net v0.0.0-20190311183353-d8887717615a => golang.org/x/net v0.0.0-20220923203811-8be639271d50

replace golang.org/x/net v0.0.0-20190404232315-eb5bcb51f2a3 => golang.org/x/net v0.0.0-20220923203811-8be639271d50

replace golang.org/x/net v0.0.0-20190620200207-3b0461eec859 => golang.org/x/net v0.0.0-20220923203811-8be639271d50

replace golang.org/x/net v0.0.0-20190827160401-ba9fcec4b297 => golang.org/x/net v0.0.0-20220923203811-8be639271d50

replace golang.org/x/net v0.0.0-20200226121028-0de0cce0169b => golang.org/x/net v0.0.0-20220923203811-8be639271d50

replace golang.org/x/net v0.0.0-20200520004742-59133d7f0dd7 => golang.org/x/net v0.0.0-20220923203811-8be639271d50

replace golang.org/x/net v0.0.0-20201021035429-f5854403a974 => golang.org/x/net v0.0.0-20220923203811-8be639271d50

replace golang.org/x/net v0.0.0-20210226172049-e18ecbb05110 => golang.org/x/net v0.0.0-20220923203811-8be639271d50

replace golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4 => golang.org/x/net v0.0.0-20220923203811-8be639271d50

replace golang.org/x/net v0.0.0-20210428140749-89ef3d95e781 => golang.org/x/net v0.0.0-20220923203811-8be639271d50

replace golang.org/x/net v0.0.0-20211015210444-4f30a5c0130f => golang.org/x/net v0.0.0-20220923203811-8be639271d50

replace golang.org/x/net v0.0.0-20211112202133-69e39bad7dc2 => golang.org/x/net v0.0.0-20220923203811-8be639271d50

replace golang.org/x/net v0.0.0-20211209124913-491a49abca63 => golang.org/x/net v0.0.0-20220923203811-8be639271d50

replace golang.org/x/net v0.0.0-20220225172249-27dd8689420f => golang.org/x/net v0.0.0-20220923203811-8be639271d50

replace golang.org/x/net v0.0.0-20220425223048-2871e0cb64e4 => golang.org/x/net v0.0.0-20220923203811-8be639271d50

replace golang.org/x/net v0.0.0-20220722155237-a158d28d115b => golang.org/x/net v0.0.0-20220923203811-8be639271d50

replace golang.org/x/text v0.3.0 => golang.org/x/text v0.3.8

replace golang.org/x/text v0.3.2 => golang.org/x/text v0.3.8

replace golang.org/x/text v0.3.3 => golang.org/x/text v0.3.8

replace golang.org/x/text v0.3.5 => golang.org/x/text v0.3.8

replace golang.org/x/text v0.3.6 => golang.org/x/text v0.3.8

replace golang.org/x/text v0.3.7 => golang.org/x/text v0.3.8

replace github.com/emicklei/go-restful v0.0.0-20170410110728-ff4f55a20633 => github.com/emicklei/go-restful/v3 v3.8.0

replace gopkg.in/yaml.v2 v2.2.1 => gopkg.in/yaml.v2 v2.2.8

replace gopkg.in/yaml.v2 v2.2.2 => gopkg.in/yaml.v2 v2.2.8

replace gopkg.in/yaml.v2 v2.2.4 => gopkg.in/yaml.v2 v2.2.8

replace gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c => gopkg.in/yaml.v3 v3.0.1

replace gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 => gopkg.in/yaml.v3 v3.0.1

replace gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b => gopkg.in/yaml.v3 v3.0.1
