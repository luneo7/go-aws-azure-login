# go-aws-azure-login

If your organization uses [Azure Active Directory](https://azure.microsoft.com) to provide SSO login to the AWS console, then there is no easy way to log in on the command line or to use the [AWS CLI](https://aws.amazon.com/cli/). This tool fixes that. It lets you use the normal Azure AD login (including MFA) from a command line to create a federated AWS session and places the temporary credentials in the proper place for the AWS CLI and SDKs.

## Installation

Installation can be done in any of the following platform - Windows, Linux, Docker, Snap

    go get https://github.com/luneo7/go-aws-azure-login

Or you can download the binary from the release page

## Usage

### Configuration

#### AWS

To configure the aws-azure-login client run:

    go-aws-azure-login -configure

You'll need your [Azure Tenant ID and the App ID URI](#getting-your-tenant-id-and-app-id-uri). To configure a named profile, use the -profile flag.

    go-aws-azure-login -configure -profile foo

##### GovCloud Support

To use aws-azure-login with AWS GovCloud, set the `region` profile property in your ~/.aws/config to the one of the GovCloud regions:

- us-gov-west-1
- us-gov-east-1

##### China Region Support

To use aws-azure-login with AWS China Cloud, set the `region` profile property in your ~/.aws/config to the China region:

- cn-north-1

#### Staying logged in, skip username/password for future logins

During the configuration you can decide to stay logged in:

    ? Stay logged in: skip authentication while refreshing aws credentials (true|false) (false)

If you set this configuration to true, the usual authentication with username/password/MFA is skipped as it's using session cookies to remember your identity. This enables you to use `-no-prompt` without the need to store your password anywhere, it's an alternative for using environment variables as described below.
As soon as you went through the full login procedure once, you can just use:

    aws-azure-login -no-prompt

or

    aws-azure-login -profile foo -no-prompt

to refresh your aws credentials.

#### Okta Support

If you use Azure AD delating to Okta, you can have a different user name and password for Okta, if you do have you can set `okta_default_username` and `okta_default_password` in the config file or in the env variable to do login with Okta without any prompt, otherwise it will prompt the username + password.

#### Environment Variables

You can optionally store your responses as environment variables:

- `AZURE_TENANT_ID`
- `AZURE_APP_ID_URI`
- `AZURE_DEFAULT_USERNAME`
- `AZURE_DEFAULT_PASSWORD`
- `AZURE_DEFAULT_ROLE_ARN`
- `AZURE_DEFAULT_DURATION_HOURS`
- `OKTA_DEFAULT_USERNAME`
- `OKTA_DEFAULT_PASSWORD`

To avoid having to `<Enter>` through the prompts after setting these environment variables, use the `--no-prompt` option when running the command.

    aws-azure-login --no-prompt

Use the `HISTCONTROL` environment variable to avoid storing the password in your bash history (notice the space at the beginning):

    $ HISTCONTROL=ignoreboth
    $  export AZURE_DEFAULT_PASSWORD=mypassword
    $ go-aws-azure-login

### Logging In

Once aws-azure-login is configured, you can log in. For the default profile, just run:

    go-aws-azure-login

You will be prompted for your username and password. If MFA is required you'll also be prompted for a verification code or mobile device approval. To log in with a named profile:

    go-aws-azure-login -profile foo

Alternatively, you can set the `AWS_PROFILE` environmental variable to the name of the profile just like the AWS CLI.

Once you log in you can use the AWS CLI or SDKs as usual!


## Automation

### Renew credentials for all configured profiles

You can renew credentials for all configured profiles in one run. This is especially useful, if the maximum session length on AWS side is configured to a low value due to security constraints. Just run:

    go-aws-azure-login -all-profiles

If you configure all profiles to stay logged in, you can easily skip the prompts:

    go-aws-azure-login -all-profiles -no-prompt

This will allow you to automate the credentials refresh procedure, eg. by running a cronjob every 5 minutes.
To skip unnecessary calls, the credentials are only getting refreshed if the time to expire is lower than 11 minutes.

## Getting Your Tenant ID and App ID URI

Your Azure AD system admin should be able to provide you with your Tenant ID and App ID URI. If you can't get it from them, you can scrape it from a login page from the myapps.microsoft.com page.

1. Load the myapps.microsoft.com page.
2. Click the chicklet for the login you want.
3. In the window the pops open quickly copy the login.microsoftonline.com URL. (If you miss it just try again. You can also open the developer console with nagivation preservation to capture the URL.)
4. The GUID right after login.microsoftonline.com/ is the tenant ID.
5. Copy the SAMLRequest URL param.
6. Paste it into a URL decoder ([like this one](https://www.samltool.com/url.php)) and decode.
7. Paste the decoded output into the a SAML deflated and encoded XML decoder ([like this one](https://www.samltool.com/decode.php)).
8. In the decoded XML output the value of the Issuer tag is the App ID URI.

## How It Works

The Azure login page uses JavaScript, which requires a real web browser. To automate this from a command line, aws-azure-login uses [Rod](https://github.com/go-rod/rod), which automates a real Chromium browser. It loads the Azure login page behind the scenes, populates your username and password (and MFA token), parses the SAML assertion, uses the [AWS STS AssumeRoleWithSAML API](http://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRoleWithSAML.html) to get temporary credentials, and saves these in the CLI credentials file.


## Support for Other Authentication Providers

Obviously, this tool only supports Azure AD as an identity provider. However, there is a lot of similarity with how other logins with other providers would work (especially if they are SAML providers). If you are interested in building support for a different provider let me know. It would be great to build a more generic AWS CLI login tool with plugins for the various providers.
