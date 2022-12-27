package forwarder

import (
	"encoding/json"
	"net/mail"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/codezombiech/aws-mail-forwarder-test/config"
	"github.com/codezombiech/aws-mail-forwarder-test/message"
)

func TestProcessMessageHeader(t *testing.T) {
	config := parseConfig(t, config.RawConfig{
		FromEmail:     "forwarder@example.com",
		SubjectPrefix: "",
		AllowPlusSign: true,
		ForwardMapping: map[string][]string{
			"private-address@example.com": {
				"private-address@example.com",
			},
		},
	})

	f := NewForwarder(config, aws.Config{})

	originalRecipientRaw := "private-address@example.com"
	originalRecipient, err := mail.ParseAddress(originalRecipientRaw)
	if err != nil {
		t.Fatalf("Invalid address: %v, %v", originalRecipientRaw, err)
	}

	reader, err := os.Open("./testdata/html-mail.mail")
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	mailMessage, err := mail.ReadMessage(reader)
	if err != nil {
		t.Fatal(err)
	}

	if err := f.processMessageHeader(mailMessage.Header, originalRecipient); err != nil {
		t.Fatal(err)
	}

	if expected, actual := "\"CodeZombie\" <forwarder@example.com>", mailMessage.Header.Get(message.FromKey); expected != actual {
		t.Errorf("From header: expected %v, got %v", expected, actual)
	}

	if expected, actual := "\"CodeZombie\" <out@example.com>", mailMessage.Header.Get(message.ToKey); expected != actual {
		t.Errorf("To header: expected %v, got %v", expected, actual)
	}

	// No *-Dkim-Signature headers
	for k := range mailMessage.Header {
		if strings.HasSuffix(k, "Dkim-Signature") {
			t.Errorf("found unexpected header %s", k)
		}
	}
}

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
