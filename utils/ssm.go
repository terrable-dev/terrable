package utils

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

func initSSMClient() (*ssm.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())

	if err != nil {
		return nil, err
	}

	ssmClient := ssm.NewFromConfig(cfg)

	return ssmClient, nil
}

func FetchSSMParameter(parameterName string) (string, error) {
	ssmClient, err := initSSMClient()
	if err != nil {
		return "", err
	}

	input := &ssm.GetParameterInput{
		Name:           aws.String(parameterName),
		WithDecryption: aws.Bool(true),
	}

	result, err := ssmClient.GetParameter(context.Background(), input)

	if err != nil {
		return "", err
	}

	return *result.Parameter.Value, nil
}
