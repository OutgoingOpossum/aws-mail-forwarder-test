//go:build e2e

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/aws/smithy-go"
	"github.com/codezombiech/aws-mail-forwarder-test/message"
)

type testMailAppResponse struct {
	Result  string                      `json:"result"`
	Message string                      `json:"message"`
	Count   int                         `json:"count"`
	Limit   int                         `json:"limit"`
	Offset  int                         `json:"offset"`
	Emails  []testMailAppEmailsResponse `json:"emails"`
}

type testMailAppEmailsResponse struct {
	DownloadUrl string `json:"downloadUrl"`
}

func sendTestMail(t *testing.T, config testConfig) {
	from := "Sender <sender@aws-mail-forwarder.org>"
	to := []string{"CI Test Forwarder <ci-test@aws-mail-forwarder.org>"}
	cc := []string{}
	bcc := []string{}

	input := sesv2.SendEmailInput{
		FromEmailAddress: aws.String(from),
		Destination: &types.Destination{
			ToAddresses:  to,
			CcAddresses:  cc,
			BccAddresses: bcc,
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{
					Data: aws.String("Test subject"),
				},
				Body: &types.Body{
					Text: &types.Content{
						Data: aws.String("Test body"),
					},
				},
			},
		},
	}

	sesClient := sesv2.NewFromConfig(config.awsConfig)

	_, err := sesClient.SendEmail(context.TODO(), &input)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			t.Fatalf("code: %s, message: %s, fault: %s", apiErr.ErrorCode(), apiErr.ErrorMessage(), apiErr.ErrorFault().String())
		} else {
			t.Fatalf(err.Error())
		}
	}
}

func receiveTestMail(t *testing.T, config testConfig) *mail.Message {
	client := http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(fmt.Sprintf(
		"https://api.testmail.app/api/json?apikey=%s&namespace=%s&tag=%s&livequery=true&timestamp_from=%d&pretty=true",
		config.testMailAppApiKey,
		config.testMailAppNamespace,
		config.testMailAppTag,
		time.Now().UTC().UnixMilli()),
	)
	if err != nil {
		t.Fatalf("Failed to receive test mail: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Receiving test mail failed with HTTP status code %v", resp.StatusCode)
	}

	bodyRaw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}
	var body testMailAppResponse
	if err := json.Unmarshal(bodyRaw, &body); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Download mail as EML file
	resp, err = http.Get(body.Emails[0].DownloadUrl)
	if err != nil {
		t.Fatalf("Failed to download mail: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Retrieving test mail failed with HTTP status code %v", resp.StatusCode)
	}

	message, err := mail.ReadMessage(resp.Body)
	if err != nil {
		t.Fatalf("Failed to parse mail: %v", err)
	}

	return message
}

func validateMail(t *testing.T, config testConfig, mail *mail.Message) {
	if expected, actual := "FORWARDER: Test subject", mail.Header.Get(message.SubjectKey); expected != actual {
		t.Errorf("expected %s, got %s", expected, actual)
	}

	if expected, actual := "\"Sender at sender@aws-mail-forwarder.org\" <ci-test@aws-mail-forwarder.org>", mail.Header.Get(message.FromKey); expected != actual {
		t.Errorf("expected %s, got %s", expected, actual)
	}

	if expected, actual := "CI Test Forwarder <ci-test@aws-mail-forwarder.org>", mail.Header.Get(message.ToKey); expected != actual {
		t.Errorf("expected %s, got %s", expected, actual)
	}
}

type testConfig struct {
	awsConfig            aws.Config
	testMailAppApiKey    string
	testMailAppNamespace string
	testMailAppTag       string
	addresses            testConfigAddresses
}

type testConfigAddresses struct {
	sender    string
	forwarder string
}

func getTestConfig(t *testing.T) testConfig {
	checkEnv(t, "AWS_REGION")
	checkEnv(t, "AWS_PROFILE")
	checkEnv(t, "TESTMAILAPP_APIKEY")
	checkEnv(t, "TESTMAILAPP_NAMESPACE")
	checkEnv(t, "TESTMAILAPP_TAG")

	awsConfig, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		t.Fatalf("Failed to load AWS config: %v", err)
	}

	return testConfig{
		awsConfig:            awsConfig,
		testMailAppApiKey:    os.Getenv("TESTMAILAPP_APIKEY"),
		testMailAppNamespace: os.Getenv("TESTMAILAPP_NAMESPACE"),
		testMailAppTag:       os.Getenv("TESTMAILAPP_TAG"),
	}
}

func checkEnv(t *testing.T, envName string) {
	if os.Getenv(envName) == "" {
		t.Fatalf("Environment variable %s must be set", envName)
	}
}

func TestForwarding(t *testing.T) {
	testConfig := getTestConfig(t)

	sendTestMail(t, testConfig)

	message := receiveTestMail(t, testConfig)

	validateMail(t, testConfig, message)
}
