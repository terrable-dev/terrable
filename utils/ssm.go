package utils

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

func GetSsmParameter(parameterName string) (string, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())

	if err != nil {
		return "", fmt.Errorf("unable to load SDK config: %v", err)
	}

	client := ssm.NewFromConfig(cfg)

	input := &ssm.GetParameterInput{
		Name:           &parameterName,
		WithDecryption: aws.Bool(true),
	}

	// Get the parameter
	output, err := client.GetParameter(context.TODO(), input)

	if err != nil {
		return "", fmt.Errorf("unable to get parameter: %v", err)
	}

	return *output.Parameter.Value, nil
}
