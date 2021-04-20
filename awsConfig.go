package main

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gopkg.in/ini.v1"
)

const timeFormat = "2006-01-02T15:04:05.000Z"

const tagName = "config"

const refreshLimitInMs int64 = 11 * 60 * 1000

type profileConfig struct {
	AzureTenantID             string  `config:"azure_tenant_id" survey:"tenantId"`
	AzureAppIDUri             string  `config:"azure_app_id_uri" survey:"appIdUri"`
	AzureDefaultUsername      string  `config:"azure_default_username" survey:"username"`
	AzureDefaultPassword      *string `config:"azure_default_password"`
	AzureDefaultRoleArn       string  `config:"azure_default_role_arn" survey:"defaultRoleArn"`
	AzureDefaultDurationHours string  `config:"azure_default_duration_hours" survey:"defaultDurationHours"`
	Region                    *string `config:"region"`
	AzureDefaultRememberMe    bool    `config:"azure_default_remember_me" survey:"rememberMe"`
	OktaDefaultUsername       *string `config:"okta_default_username" survey:"oktaUsername"`
	OktaDefaultPassword       *string `config:"okta_default_password" survey:"oktaPassword"`
}

type profileCredentials struct {
	AwsAccessKeyID     string `config:"aws_access_key_id"`
	AwsSecretAccessKey string `config:"aws_secret_access_key"`
	AwsSessionToken    string `config:"aws_session_token"`
	AwsExpiration      string `config:"aws_expiration"`
}

func setProfileConfig(profileName string, values profileConfig) {
	sectionName := getSectionName(profileName)

	config := load(CONFIG)
	section := config.Section(sectionName)

	setSectionValues(section, values)

	save(CONFIG, config)
}

func getProfileConfig(profileName string) profileConfig {
	sectionName := getSectionName(profileName)

	config := load(CONFIG)

	section := config.Section(sectionName)

	azureDefaultRememberMe, err := strconv.ParseBool(section.Key("azure_default_remember_me").Value())

	if err != nil {
		azureDefaultRememberMe = false
	}

	return profileConfig{
		AzureTenantID:             section.Key("azure_tenant_id").Value(),
		AzureAppIDUri:             section.Key("azure_app_id_uri").Value(),
		AzureDefaultUsername:      section.Key("azure_default_username").Value(),
		AzureDefaultPassword:      stringToPointer(section.Key("azure_default_password").Value()),
		AzureDefaultRoleArn:       section.Key("azure_default_role_arn").Value(),
		AzureDefaultDurationHours: section.Key("azure_default_duration_hours").Value(),
		Region:                    stringToPointer(section.Key("region").Value()),
		AzureDefaultRememberMe:    azureDefaultRememberMe,
		OktaDefaultUsername:       stringToPointer(section.Key("okta_default_username").Value()),
		OktaDefaultPassword:       stringToPointer(section.Key("okta_default_password").Value()),
	}
}

func isProfileAboutToExpire(profileName string) bool {
	config := load(CREDENTIALS)

	section := config.Section(profileName)

	aws_expiration := section.Key("aws_expiration").Value()

	expirationDate := time.Now()

	if aws_expiration != "" {
		var err error
		expirationDate, err = time.Parse(timeFormat, aws_expiration)
		if err != nil {
			fmt.Printf("Invalid profile expiration: %v", err)
			os.Exit(1)
		}
	}

	timeDifference := time.Until(expirationDate)

	return timeDifference.Milliseconds() < refreshLimitInMs
}

func setProfileCredentials(profileName string, values profileCredentials) {
	config := load(CREDENTIALS)
	section := config.Section(profileName)

	setSectionValues(section, values)

	save(CREDENTIALS, config)
}

func getAllProfileNames() []string {
	config := load(CONFIG)

	sections := config.Sections()

	var profiles []string

	for _, section := range sections {
		if section != nil {
			profiles = append(profiles, strings.ReplaceAll(section.Name(), "profile ", ""))
		}
	}

	return profiles
}

func getSectionName(profileName string) string {
	sectionName := "default"
	if profileName != "default" {
		sectionName = fmt.Sprintf("profile %s", profileName)
	}
	return sectionName
}

func setSectionValues(section *ini.Section, values interface{}) {
	v := reflect.ValueOf(values)
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		tag := field.Tag.Get(tagName)

		if value.Kind() == reflect.Ptr {
			if value.IsNil() || value.Elem().Interface().(string) == "" {
				section.DeleteKey(tag)
			} else {
				section.NewKey(tag, value.Elem().Interface().(string))
			}
		} else {
			if value.Kind() == reflect.Bool {
				section.NewKey(tag, strconv.FormatBool(value.Interface().(bool)))
			} else {
				section.NewKey(tag, value.Interface().(string))
			}
		}
	}
}

func load(pathType PathType) *ini.File {
	p, ok := paths[pathType]
	if !ok {
		fmt.Printf("Unkown config path type: %v", pathType)
		os.Exit(1)
	}

	cfg, err := ini.Load(p)
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}

	return cfg
}

func save(pathType PathType, data *ini.File) {
	p, ok := paths[pathType]
	if !ok {
		fmt.Printf("Unkown config path type: %v", pathType)
		os.Exit(1)
	}

	if data == nil {
		fmt.Printf("You must provide a data for saving")
		os.Exit(1)
	}

	data.SaveTo(p)
}

func stringToPointer(v string) *string {
	if v != "" {
		return &v
	}
	return nil
}
