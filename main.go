package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	profile      string
	allProfiles  bool
	forceRefresh bool
	configure    bool
	mode         string
	noVerifySSL  bool
	noPrompt     bool
)

func init() {
	const (
		profileDefaultValue      = ""
		profileUsage             = "The name of the profile to log in with (or configure)"
		allProfilesDefaultValue  = false
		allProfilesUsage         = "Run for all configured profiles"
		forceRefreshDefaultValue = false
		forceRefreshUsage        = "Force a credential refresh, even if they are still valid"
		configureDefaultValue    = false
		configureUsage           = "Configure the profile"
		modeDefaultValue         = "cli"
		modeUsage                = "'cli' to hide the login page and perform the login through the CLI (default behavior), 'gui' to perform the login through the Azure GUI (more reliable but only works on GUI operating system), 'debug' to show the login page but perform the login through the CLI (useful to debug issues with the CLI login)"
		noVerifySSLDefaultValue  = false
		noVerifySSLUsage         = "Disable SSL Peer Verification for connections to AWS (no effect if behind proxy)"
		noPromptDefaultValue     = false
		noPromptUsage            = "Do not prompt for input and accept the default choice"
	)

	flag.StringVar(&profile, "profile", profileDefaultValue, profileUsage)
	flag.StringVar(&profile, "p", profileDefaultValue, profileUsage+" (shorthand)")
	flag.BoolVar(&allProfiles, "all-profiles", allProfilesDefaultValue, allProfilesUsage)
	flag.BoolVar(&allProfiles, "a", allProfilesDefaultValue, allProfilesUsage+" (shorthand)")
	flag.BoolVar(&forceRefresh, "force-refresh", forceRefreshDefaultValue, forceRefreshUsage)
	flag.BoolVar(&forceRefresh, "f", forceRefreshDefaultValue, forceRefreshUsage+" (shorthand)")
	flag.StringVar(&mode, "mode", modeDefaultValue, modeUsage)
	flag.StringVar(&mode, "m", modeDefaultValue, modeUsage+" (shorthand)")
	flag.BoolVar(&configure, "configure", configureDefaultValue, configureUsage)
	flag.BoolVar(&configure, "c", configureDefaultValue, configureUsage+" (shorthand)")
	flag.BoolVar(&noVerifySSL, "no-verify-ssl", noVerifySSLDefaultValue, noVerifySSLUsage)
	flag.BoolVar(&noPrompt, "no-prompt", noPromptDefaultValue, noPromptUsage)

	flag.Parse()
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: Unused command line arguments detected.\n")
		flag.Usage()
		os.Exit(2)
	}
}

func main() {
	var profileName string

	if profile != "" {
		profileName = profile
	} else if osAWSProfile := os.Getenv("AWS_PROFILE"); osAWSProfile != "" {
		profileName = osAWSProfile
	} else {
		profileName = "default"
	}

	if configure {
		configureProfile(profileName)
	} else {
		if allProfiles {
			loginAll(forceRefresh, noVerifySSL, noPrompt)

		} else {
			login(profileName, noVerifySSL, noPrompt)
		}
	}

}
