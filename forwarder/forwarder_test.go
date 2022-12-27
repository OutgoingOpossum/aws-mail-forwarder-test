package forwarder

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/codezombiech/aws-mail-forwarder-test/config"
)

func NotTestForward(t *testing.T) {
	config := parseConfig(t, config.RawConfig{
		FromEmail:     "forwarder@example.com",
		SubjectPrefix: "",
		S3: config.S3Config{
			BucketName: "s3-bucket-name",
			Incoming: config.S3IncomingConfig{
				NewPrefix:       "in/new/",
				SpamVirusPrefix: "in/spam-virus/",
				ForwardedPrefix: "in/forwarded/",
				FailedPrefix:    "in/failed/",
			},
			Outgoing: config.S3OutgoingConfig{
				SentPrefix:   "out/sent/",
				FailedPrefix: "out/failed/",
			},
		},
		AllowPlusSign: true,
		ForwardMapping: map[string][]string{
			"lambda@amazon.com": {
				"lambda@example.com",
			},
		},
	})

	event := events.SimpleEmailEvent{}
	bytes, err := os.ReadFile("testdata/ses-lambda-event.json")
	if err != nil {
		t.Fatal(err)
	}
	json.Unmarshal(bytes, &event)

	sesEvent := event.Records[0].SES

	forwarder := NewForwarder(config, aws.Config{})
	err = forwarder.Forward(sesEvent)
	if err != nil {
		t.Fatal(err)
	}
}

func parseConfig(t *testing.T, rawConfig config.RawConfig) *config.ParsedConfig {
	config, err := config.ParseConfig(&rawConfig)
	if err != nil {
		t.Fatal(err)
	}
	return config
}
