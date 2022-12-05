package main

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientauthv1beta1 "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"
)

const (
	kindExecCredential     = "ExecCredential"
	presignedURLExpiration = 15 * time.Minute
	clusterIDHeader        = "x-k8s-aws-id"
	v1Prefix               = "k8s-aws-v1."
)

type Config struct {
	ClusterID string
	RoleARN   string
	Region    string
}

var conf Config

func init() {
	flag.StringVar(&conf.Region, "region", "", "")
	flag.StringVar(&conf.ClusterID, "cluster-name", "", "")
	flag.StringVar(&conf.RoleARN, "role-arn", "", "")
	flag.Parse()
}

func validateConfig(c Config) error {
	if c.ClusterID == "" {
		return errors.New("cluster-name cannot be empty")
	}
	return nil
}

func getToken(c Config) string {
	cfg, err := config.LoadDefaultConfig(context.TODO())

	if c.Region != "" {
		cfg.Region = *aws.String(c.Region)
	}

	if c.RoleARN != "" {
		client := sts.NewFromConfig(cfg)
		cfg.Credentials = stscreds.NewAssumeRoleProvider(client, c.RoleARN)
	}

	apiOption := sts.WithAPIOptions(SetHttpHeader(clusterIDHeader, c.ClusterID))
	presignClient := sts.NewPresignClient(sts.NewFromConfig(cfg, apiOption))

	getCallerIdentity, err := presignClient.PresignGetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		panic(err)
	}

	return v1Prefix + b64.RawURLEncoding.EncodeToString([]byte(getCallerIdentity.URL))
}

func formatJSON(token string) []byte {
	expTime := time.Now().Add(presignedURLExpiration - 1*time.Minute)
	expirationTimestamp := v1.NewTime(expTime)

	apiVersion := clientauthv1beta1.SchemeGroupVersion.String()
	execObj := &clientauthv1beta1.ExecCredential{
		TypeMeta: v1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       kindExecCredential,
		},
		Status: &clientauthv1beta1.ExecCredentialStatus{
			ExpirationTimestamp: &expirationTimestamp,
			Token:               token,
		},
	}

	jsonData, err := json.Marshal(execObj)
	if err != nil {
		panic(err)
	}

	return jsonData
}
func main() {
	err := validateConfig(conf)
	if err != nil {
		panic(err)
	}
	token := getToken(conf)
	fmt.Println(string(formatJSON(token)))

	return
}

func SetHttpHeader(key, value string) func(*middleware.Stack) error {
	return func(stack *middleware.Stack) error {
		return stack.Build.Add(middleware.BuildMiddlewareFunc("EKSSetHeader", func(
			ctx context.Context, in middleware.BuildInput, next middleware.BuildHandler,
		) (
			middleware.BuildOutput, middleware.Metadata, error,
		) {
			switch v := in.Request.(type) {
			case *smithyhttp.Request:
				v.Header.Add(key, value)
			}
			return next.HandleBuild(ctx, in)
		}), middleware.Before)
	}
}
