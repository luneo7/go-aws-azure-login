module github.com/luneo7/go-aws-azure-login

go 1.17

require (
	github.com/AlecAivazis/survey/v2 v2.3.6
	github.com/aws/aws-sdk-go-v2/config v1.9.0
	github.com/aws/aws-sdk-go-v2/service/sts v1.8.0
	github.com/go-rod/rod v0.101.8
	github.com/google/uuid v1.3.0
	gopkg.in/ini.v1 v1.63.2
)

require (
	github.com/aws/aws-sdk-go-v2 v1.10.0 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.5.0 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.7.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.2.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.4.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.5.0 // indirect
	github.com/aws/smithy-go v1.8.1 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/ysmood/goob v0.3.0 // indirect
	github.com/ysmood/gson v0.6.4 // indirect
	github.com/ysmood/leakless v0.7.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/term v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
)

replace github.com/go-rod/rod => github.com/luneo7/rod v0.101.9-0.20220131185440-7433d1ef4c0a
