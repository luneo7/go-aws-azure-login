package main

import (
	"bytes"
	"compress/flate"
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/google/uuid"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

const (
	AZURE_AD_SSO          = "autologon.microsoftazuread-sso.com"
	AWS_SAML_ENDPOINT     = "https://signin.aws.amazon.com/saml"
	AWS_CN_SAML_ENDPOINT  = "https://signin.amazonaws.cn/saml"
	AWS_GOV_SAML_ENDPOINT = "https://signin.amazonaws-us-gov.com/saml"

	WIDTH  = 425
	HEIGHT = 550
)

type state struct {
	name     string
	selector string
	handler  func(pg *rod.Page, el *rod.Element, noPrompt bool, defaultUserName string, defaultUserPassword *string, defaultOktaUserName *string, defaultOktaPassword *string)
}

type samlResponse struct {
	XMLName   xml.Name
	Assertion samlAssertion `xml:"Assertion"`
}

type samlAssertion struct {
	XMLName            xml.Name
	AttributeStatement samlAttributeStatement
}

type samlAttributeValue struct {
	XMLName xml.Name
	Type    string `xml:"xsi:type,attr"`
	Value   string `xml:",innerxml"`
}

type samlAttribute struct {
	XMLName         xml.Name
	Name            string               `xml:",attr"`
	AttributeValues []samlAttributeValue `xml:"AttributeValue"`
}

type samlAttributeStatement struct {
	XMLName    xml.Name
	Attributes []samlAttribute `xml:"Attribute"`
}

type role struct {
	roleArn      string
	principalArn string
}

var states = []state{
	{
		name:     "username input",
		selector: `input[name="loginfmt"]:not(.moveOffScreen)`,
		handler: func(pg *rod.Page, el *rod.Element, noPrompt bool, defaultUserName string, _ *string, _ *string, _ *string) {
			username := defaultUserName

			if !noPrompt {
				prompt := &survey.Input{
					Message: "Azure Username:",
					Default: defaultUserName,
				}
				survey.AskOne(prompt, &username, survey.WithValidator(survey.Required))
			}

			el.MustWaitVisible()
			el.MustSelectAllText().MustInput("")
			el.MustInput(strings.TrimSpace(username))

			sb := pg.MustElement(`input[type=submit]`)

			sb.MustWaitVisible()
			wait := pg.MustWaitRequestIdle()
			sb.MustClick()
			wait()

			pContext := pg.GetContext()
			defer func() {
				pg.Context(pContext)
			}()

			ctx, cancel := context.WithCancel(pContext)
			defer cancel()

			ch := make(chan bool, 1)

			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					default:
						_, err := pg.Sleeper(rod.NotFoundSleeper).Element("input[name=loginfmt]")
						if err != nil {
							ch <- true
							return
						}
					}
				}
			}()

			go func() {
				pg.Timeout(20 * time.Second).Race().
					Element("input[name=loginfmt].has-error").
					Element("input[name=loginfmt].moveOffScreen").
					Element("input[name=loginfmt]").Handle(func(e *rod.Element) error {
					return e.WaitInvisible()
				}).Do()

				select {
				case <-ctx.Done():
					return
				default:
					ch <- true
					return
				}
			}()

			select {
			case <-ch:
			case <-time.After(25 * time.Second):
			}
		},
	},
	{
		name:     "password input",
		selector: `input[name="Password"]:not(.moveOffScreen),input[name="passwd"]:not(.moveOffScreen)`,
		handler: func(pg *rod.Page, el *rod.Element, noPrompt bool, _ string, defaultUserPassword *string, _ *string, _ *string) {
			alert, err := pg.Sleeper(rod.NotFoundSleeper).Element(".alert-error")

			if alert != nil && err == nil {
				fmt.Println(alert.Text())
			}

			var password string

			if noPrompt && defaultUserPassword != nil {
				password = *defaultUserPassword
			} else {
				prompt := &survey.Password{
					Message: "Azure Password",
				}
				survey.AskOne(prompt, &password, survey.WithValidator(survey.Required))
			}

			el.MustWaitVisible()
			el.MustSelectAllText().MustInput("")
			el.MustInput(password)

			wait := pg.MustWaitRequestIdle()
			pg.MustElement("span[class=submit],input[type=submit]").MustClick()
			wait()

			time.Sleep(time.Millisecond * 500)
		},
	},
	{
		name:     "OKTA username/password input",
		selector: `form:not(.o-form-saving) > div span.okta-form-input-field input[autocomplete="username"]:not([disabled])`,
		handler: func(pg *rod.Page, el *rod.Element, noPrompt bool, defaultUserName string, defaultUserPassword *string, defaultOktaUserName *string, defaultOktaPassword *string) {
			alert, err := pg.Sleeper(rod.NotFoundSleeper).Element(`div[role="alert"]`)

			if alert != nil && err == nil {
				t, _ := alert.Text()
				if t != "" {
					fmt.Println(t)
				}
			}

			var username string
			var password string
			shouldAskPassword := true

			if noPrompt {
				if defaultOktaUserName != nil {
					username = *defaultOktaUserName
				} else {
					username = defaultUserName
				}

				if defaultOktaPassword != nil {
					password = *defaultOktaPassword
					shouldAskPassword = false
				} else if defaultUserPassword != nil {
					password = *defaultUserPassword
					shouldAskPassword = false
				}
			} else {
				defUser := defaultUserName
				if defaultOktaUserName != nil {
					defUser = *defaultOktaUserName
				}
				promptUsername := &survey.Input{
					Message: "Okta Username:",
					Default: defUser,
				}
				survey.AskOne(promptUsername, &username, survey.WithValidator(survey.Required))
			}

			el.MustWaitVisible()
			el.MustSelectAllText().MustInput("")
			el.MustInput(username)

			if shouldAskPassword {
				promptPasswd := &survey.Password{
					Message: "Okta Password:",
				}
				survey.AskOne(promptPasswd, &password, survey.WithValidator(survey.Required))
			}

			time.Sleep(time.Millisecond * 500)

			pwdEl := pg.MustElement(`input[type="password"]`)
			pwdEl.MustWaitVisible()
			pwdEl.MustSelectAllText().MustInput("")
			pwdEl.MustInput(password)

			time.Sleep(time.Millisecond * 500)

			btn, err := pg.Sleeper(rod.NotFoundSleeper).Element(`input:not([disabled]):not(.link-button-disabled):not(.btn-disabled)[type=submit]`)
			if err == nil {
				wait := pg.MustWaitRequestIdle()
				btn.MustClick()
				wait()
				time.Sleep(time.Millisecond * 500)
			}
		},
	},
	{
		name:     "OKTA SELECT PUSH Form",
		selector: `div[data-se="okta_verify-push"] > a:not([disabled]):not(.link-button-disabled):not(.btn-disabled)`,
		handler: func(pg *rod.Page, el *rod.Element, _ bool, defaultUserName string, _ *string, _ *string, _ *string) {
			alert, err := pg.Sleeper(rod.NotFoundSleeper).Element(".infobox-error")

			if alert != nil && err == nil {
				t, _ := alert.Text()
				if t != "" {
					fmt.Println(t)
				}
			}

			btn, err := pg.Sleeper(rod.NotFoundSleeper).Element(`div[data-se="okta_verify-push"] > a:not([disabled]):not(.btn-disabled):not(.link-button-disabled)`)
			if err == nil && btn != nil {
				btn.MustWaitVisible()
				wait := pg.MustWaitRequestIdle()
				btn.MustClick()
				wait()
				time.Sleep(time.Millisecond * 500)
			}
		},
	},
	{
		name:     "OKTA DO PUSH Form",
		selector: `a.send-push:not([disabled]):not(.link-button-disabled):not(.btn-disabled)`,
		handler: func(pg *rod.Page, el *rod.Element, _ bool, defaultUserName string, _ *string, _ *string, _ *string) {
			alert, err := pg.Sleeper(rod.NotFoundSleeper).Element(".infobox-error")

			if alert != nil && err == nil {
				t, _ := alert.Text()
				if t != "" {
					fmt.Println(t)
				}
			}

			btn, err := pg.Sleeper(rod.NotFoundSleeper).Element(`a.send-push:not([disabled]):not(.btn-disabled):not(.link-button-disabled)`)
			if err == nil && btn != nil {
				btn.MustWaitVisible()
				wait := pg.MustWaitRequestIdle()
				btn.MustClick()
				wait()
				time.Sleep(time.Millisecond * 500)
			}
		},
	},
}

func loadProfileFromEnv() profileConfig {
	envVars := []string{
		"azure_tenant_id",
		"azure_app_id_uri",
		"azure_default_username",
		"azure_default_password",
		"azure_default_role_arn",
		"azure_default_duration_hours",
		"region",
		"okta_default_username",
		"okta_default_password",
	}

	profile := profileConfig{}

	v := reflect.ValueOf(&profile).Elem()
	t := v.Type()

	for _, envVar := range envVars {
		val, exists := os.LookupEnv(envVar)
		if exists {
			for i := 0; i < v.NumField(); i++ {
				tag := t.Field(i).Tag.Get(tagName)
				if tag == envVar {
					f := v.Field(i)
					if f.Kind() == reflect.String {
						f.SetString(val)
					} else if f.Kind() == reflect.Ptr {
						f.Set(reflect.ValueOf(&val))
					}
				}
			}
		}
	}

	return profile
}

func loadProfile(profileName string) profileConfig {
	profile := getProfileConfig(profileName)
	envProfile := loadProfileFromEnv()

	if (envProfile != profileConfig{}) {
		v := reflect.ValueOf(&profile).Elem()
		vEnv := reflect.ValueOf(&envProfile).Elem()
		t := v.Type()

		for i := 0; i < t.NumField(); i++ {
			value := v.Field(i)
			envValue := vEnv.Field(i)

			if envValue.Kind() == reflect.Ptr {
				if !envValue.IsNil() {
					value.Set(reflect.ValueOf(envValue.Interface()))
				}
			} else if envValue.Kind() == reflect.Bool {
				if value.Interface().(bool) != envValue.Interface().(bool) {
					value.SetBool(envValue.Interface().(bool))
				}
			} else {
				nValue := envValue.Interface().(string)
				if value.Interface().(string) != nValue && nValue != "" {
					value.SetString(envValue.Interface().(string))
				}
			}
		}
	}

	return profile
}

func login(
	profileName string,
	awsNoVerifySsl bool,
	noPrompt bool) {

	profile := loadProfile(profileName)

	assertionConsumerServiceURL := AWS_SAML_ENDPOINT

	if profile.Region != nil {
		if strings.HasPrefix(*profile.Region, "us-gov") {
			assertionConsumerServiceURL = AWS_GOV_SAML_ENDPOINT
		} else if strings.HasPrefix(*profile.Region, "cn-") {
			assertionConsumerServiceURL = AWS_CN_SAML_ENDPOINT
		}
	}

	loginUrl := createLoginUrl(profile.AzureAppIDUri, profile.AzureTenantID, assertionConsumerServiceURL)

	saml := performLogin(loginUrl, noPrompt, profile.AzureDefaultUsername, profile.AzureDefaultPassword, profile.OktaDefaultUsername, profile.OktaDefaultPassword)

	roles := parseRolesFromSamlResponse(saml)

	rl, durationHours := askUserForRoleAndDuration(roles, noPrompt, profile.AzureDefaultRoleArn, profile.AzureDefaultDurationHours)

	assumeRole(profileName, saml, rl, durationHours, awsNoVerifySsl, profile.Region)
}

func loginAll(forceRefresh bool, awsNoVerifySsl bool, noPrompt bool) {
	allProfiles := getAllProfileNames()

	for _, profileName := range allProfiles {
		if !forceRefresh && !isProfileAboutToExpire(profileName) {
			continue
		}

		login(profileName, awsNoVerifySsl, noPrompt)
	}
}

func createLoginUrl(appIDUri string, tenantID string, assertionConsumerServiceURL string) string {
	id := uuid.NewString()

	samlRequest := fmt.Sprintf(`
	<samlp:AuthnRequest xmlns="urn:oasis:names:tc:SAML:2.0:metadata" ID="id%s" Version="2.0" IssueInstant="%s" IsPassive="false" AssertionConsumerServiceURL="%s" xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol">
		<Issuer xmlns="urn:oasis:names:tc:SAML:2.0:assertion">%s</Issuer>
		<samlp:NameIDPolicy Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"></samlp:NameIDPolicy>
	</samlp:AuthnRequest>
	`, id, time.Now().Format(time.RFC3339), assertionConsumerServiceURL, appIDUri)

	var buffer bytes.Buffer

	flateWriter, _ := flate.NewWriter(&buffer, -1)

	flateWriter.Write([]byte(samlRequest))
	flateWriter.Flush()
	flateWriter.Close()

	samlBase64 := base64.StdEncoding.EncodeToString(buffer.Bytes())

	return fmt.Sprintf("https://login.microsoftonline.com/%s/saml2?SAMLRequest=%s", tenantID, url.QueryEscape(samlBase64))
}

func performLogin(urlString string, noPrompt bool, defaultUserName string, defaultUserPassword *string, defaultOktaUserName *string, defaultOktaPassword *string) string {
	launcher.SetDefaultHosts([]launcher.Host{launcher.HostGoogle})

	browser := rod.New().MustConnect()
	defer browser.MustClose()

	router := browser.HijackRequests()
	defer router.MustStop()

	samlResponseChan := make(chan string, 1)
	samlResponse := ""

	router.MustAdd("https://*amazon*", func(ctx *rod.Hijack) {
		reqURL := ctx.Request.URL().String()

		if reqURL == AWS_SAML_ENDPOINT || reqURL == AWS_GOV_SAML_ENDPOINT || reqURL == AWS_CN_SAML_ENDPOINT {

			val, err := url.ParseQuery(ctx.Request.Body())

			if err != nil {
				fmt.Printf("Fail to saml endpoint response: %v", err)
				os.Exit(1)
			}

			samlResponseChan <- val.Get("SAMLResponse")

			ctx.Response.Fail(proto.NetworkErrorReasonInternetDisconnected)
		} else {
			ctx.ContinueRequest(&proto.FetchContinueRequest{})
		}
	})

	go router.Run()

	page := browser.MustPage()
	wait := page.WaitNavigation(proto.PageLifecycleEventNameDOMContentLoaded)
	page.MustNavigate(urlString)
	wait()

Loop:
	for {
		for _, st := range states {
			select {
			case x, ok := <-samlResponseChan:
				if ok {
					samlResponse = x
					break Loop
				}
			default:
			}

			el, err := page.Sleeper(rod.NotFoundSleeper).Element(st.selector)

			if err == nil {
				st.handler(page, el, noPrompt, defaultUserName, defaultUserPassword, defaultOktaUserName, defaultOktaPassword)
			}
		}
	}

	return samlResponse
}

func parseRolesFromSamlResponse(assertion string) []role {
	b64, err := base64.StdEncoding.DecodeString(assertion)

	if err != nil {
		fmt.Printf("Fail to parse roles: %v", err)
		os.Exit(1)
	}

	var roles []role
	var sResponse samlResponse

	err = xml.Unmarshal(b64, &sResponse)

	if err != nil {
		fmt.Printf("Fail to unmarshal roles: %v", err)
		os.Exit(1)
	}

	for _, attr := range sResponse.Assertion.AttributeStatement.Attributes {
		if attr.Name == "https://aws.amazon.com/SAML/Attributes/Role" {
			for _, val := range attr.AttributeValues {
				parts := strings.Split(val.Value, ",")

				if strings.Contains(parts[0], ":role/") {
					roles = append(roles, role{
						roleArn:      strings.TrimSpace(parts[0]),
						principalArn: strings.TrimSpace(parts[1]),
					})
				} else {
					roles = append(roles, role{
						roleArn:      strings.TrimSpace(parts[1]),
						principalArn: strings.TrimSpace(parts[0]),
					})
				}

			}
		}
	}

	return roles
}

func askUserForRoleAndDuration(
	roles []role,
	noPrompt bool,
	defaultRoleArn string,
	defaultDurationHours string) (r role, durationHours int32) {
	durationHoursP, _ := strconv.ParseInt(defaultDurationHours, 10, 32)
	durationHours = int32(durationHoursP)

	if len(roles) == 0 {
		fmt.Println("No roles found in SAML response.")
		os.Exit(1)
	} else if len(roles) == 1 {
		r = roles[0]
	} else {
		if noPrompt && defaultRoleArn != "" {
			for _, rl := range roles {
				if rl.roleArn == defaultRoleArn {
					r = rl
					break
				}
			}
		}

		if (role{} == r) {
			var options []string

			for _, rl := range roles {
				options = append(options, rl.roleArn)
			}

			rArn := ""
			prompt := &survey.Select{
				Message: "Role:",
				Options: options,
				Default: defaultRoleArn,
			}
			survey.AskOne(prompt, &rArn, survey.WithValidator(survey.Required))

			for _, rl := range roles {
				if rl.roleArn == rArn {
					r = rl
					break
				}
			}
		}
	}

	if !(noPrompt && defaultDurationHours != "") {
		inp := &survey.Input{Message: "Session Duration Hours (up to 12):", Default: defaultDurationHours}
		hq := ""

		survey.AskOne(inp, &hq, survey.WithValidator(func(val interface{}) error {
			if str, ok := val.(string); !ok {
				return errors.New("invalid number")
			} else if n, err := strconv.ParseInt(str, 10, 32); err != nil || n <= 0 || n > 12 {
				return errors.New("duration hours must be between 0 and 12")
			}
			return nil
		}))

		durationHoursP, _ = strconv.ParseInt(hq, 10, 32)
		durationHours = int32(durationHoursP)
	}
	return
}

func assumeRole(
	profileName string,
	assertion string,
	role role,
	durationHours int32,
	awsNoVerifySsl bool,
	region *string) {

	durationSeconds := durationHours * 60 * 60
	stsInput := sts.AssumeRoleWithSAMLInput{
		PrincipalArn:    &role.principalArn,
		RoleArn:         &role.roleArn,
		SAMLAssertion:   &assertion,
		DurationSeconds: &durationSeconds,
	}

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		fmt.Printf("Fail to get AWS config: %v", err)
		os.Exit(1)
	}

	if region != nil {
		cfg.Region = *region
	}

	stsClient := sts.NewFromConfig(cfg)

	stsResult, err := stsClient.AssumeRoleWithSAML(context.Background(), &stsInput)

	if err != nil {
		fmt.Printf("Fail to assume role: %v", err)
		os.Exit(1)
	}

	setProfileCredentials(profileName,
		profileCredentials{
			AwsAccessKeyID:     *stsResult.Credentials.AccessKeyId,
			AwsSecretAccessKey: *stsResult.Credentials.SecretAccessKey,
			AwsSessionToken:    *stsResult.Credentials.SessionToken,
			AwsExpiration:      (*stsResult.Credentials.Expiration).Format(timeFormat),
		},
	)
}
