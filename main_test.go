package main

import (
	"os"
	"testing"
	"encoding/base64"
	"strings"
	"net/url"
	"github.com/stretchr/testify/assert"
)

type Token struct {
	ClusterID bool 
	Region string
}

func setAWSCreds() {
	os.Setenv("AWS_REGION", "af-south-1")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "CLIENTID")
	os.Setenv("AWS_ACCESS_KEY_ID", "KEYID")
}

func unsetAWSCreds() {
	os.Unsetenv("AWS_REGION")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	os.Unsetenv("AWS_ACCESS_KEY_ID")
}

func parseToken(t string) (*Token, error){
	parsed := Token{}
	tokenBytes, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(t, v1Prefix))
	if err != nil {
		return nil, err
	}

	parsedURL, err := url.Parse(string(tokenBytes))
	if err != nil {
		return nil, err
	}
	parsed.Region = strings.Split(parsedURL.Host, ".")[1]

	paramsValues := make(url.Values)
	queryParams, err := url.ParseQuery(parsedURL.RawQuery)
	for key, values := range queryParams {
		paramsValues.Set(strings.ToLower(key), values[0])
	}

	signedHeaders := strings.Split(paramsValues.Get("x-amz-signedheaders"), ";")
	for _, hdr := range signedHeaders{
		if strings.ToLower(hdr) == strings.ToLower(clusterIDHeader) {
			parsed.ClusterID = true
		}
	}

	return &parsed, nil
}

func hasPrefix(s string) bool {
	return strings.HasPrefix(s, v1Prefix)
}

func TestGetToken(t *testing.T) {
	conf := Config{
		ClusterID: "my-cluster", 
	}
	setAWSCreds()

	token, err := getToken(conf)
	assert.NoError(t, err)
	assert.True(t, hasPrefix(token))

	tokenParsed, err := parseToken(token)
	assert.NoError(t, err)
	unsetAWSCreds()
	assert.Equal(t, tokenParsed.Region, "af-south-1")
	assert.True(t, tokenParsed.ClusterID)
}

func TestCanOverrideRegion(t *testing.T) {
	conf := Config{
		Region: "eu-west-1",
		ClusterID: "my-cluster", 
	}
	setAWSCreds()

	token, err := getToken(conf)
	assert.NoError(t, err)
	p, err := parseToken(token)
	assert.NoError(t, err)
	assert.Equal(t, p.Region, conf.Region)
}
