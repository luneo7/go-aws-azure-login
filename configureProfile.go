package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/AlecAivazis/survey/v2"
)

func configureProfile(profileName string) {
	profile := getProfileConfig(profileName)

	var qs = []*survey.Question{
		{
			Name:     "tenantId",
			Prompt:   &survey.Input{Message: "Azure Tenant ID:", Default: profile.AzureTenantID},
			Validate: survey.Required,
		},
		{
			Name:     "appIdUri",
			Prompt:   &survey.Input{Message: "Azure App ID URI:", Default: profile.AzureAppIDUri},
			Validate: survey.Required,
		},
		{
			Name:   "username",
			Prompt: &survey.Input{Message: "Default Azure Username:", Default: profile.AzureDefaultUsername},
		},
		{
			Name:   "oktaUsername",
			Prompt: &survey.Input{Message: "Default Okta Username:", Default: stringPointerToString(profile.OktaDefaultUsername)},
			Transform: func(ans interface{}) interface{} {
				if str, ok := ans.(string); ok {
					if str != "" {
						return &str
					}
					return nil
				}
				return nil
			},
		},
		{
			Name:     "rememberMe",
			Prompt:   &survey.Confirm{Message: "Stay logged in: skip authentication while refreshing aws credentials", Default: profile.AzureDefaultRememberMe},
			Validate: survey.Required,
		},
		{
			Name:   "defaultRoleArn",
			Prompt: &survey.Input{Message: "Default Role ARN (if multiple):", Default: profile.AzureDefaultRoleArn},
		},
		{
			Name:   "defaultDurationHours",
			Prompt: &survey.Input{Message: "Default Session Duration Hours (up to 12):", Default: profile.AzureDefaultDurationHours},
			Validate: func(val interface{}) error {
				if str, ok := val.(string); !ok {
					return errors.New("invalid number")
				} else if n, err := strconv.ParseInt(str, 10, 64); err != nil || n <= 0 || n > 12 {
					return errors.New("duration hours must be between 0 and 12")
				}
				return nil
			},
		},
	}

	err := survey.Ask(qs, &profile)
	if err != nil {
		fmt.Printf("Fail to get profile answers: %v", err)
		os.Exit(1)
	}

	setProfileConfig(profileName, profile)
}

func stringPointerToString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
