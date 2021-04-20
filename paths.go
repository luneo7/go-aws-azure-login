package main

import (
	"os"
	"path/filepath"
)

type PathType string

const (
	AWSDIR      PathType = "awsDir"
	CONFIG      PathType = "config"
	CREDENTIALS PathType = "credentials"
	CHROMIUM    PathType = "chromium"
)

var userHomeDir, _ = os.UserHomeDir()
var awsDir = filepath.Join(userHomeDir, ".aws")

var paths = map[PathType]string{
	AWSDIR:      awsDir,
	CONFIG:      ifThenElse(os.Getenv("AWS_CONFIG_FILE") != "", os.Getenv("AWS_CONFIG_FILE"), filepath.Join(awsDir, string(CONFIG))),
	CREDENTIALS: ifThenElse(os.Getenv("AWS_SHARED_CREDENTIALS_FILE") != "", os.Getenv("AWS_CONFIG_FILE"), filepath.Join(awsDir, string(CREDENTIALS))),
	CHROMIUM:    filepath.Join(awsDir, string(CHROMIUM)),
}

func ifThenElse(condition bool, a string, b string) string {
	if condition {
		return a
	}
	return b
}
