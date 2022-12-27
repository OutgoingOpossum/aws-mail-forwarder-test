package message

import (
	"net/mail"
	"os"
	"strings"
	"testing"

	"github.com/codezombiech/aws-mail-forwarder-test/config"
	"github.com/google/go-cmp/cmp"
)

func TestProcessMessageHeaderReplyTo(t *testing.T) {
	tests := map[string]struct {
		from    string
		replyTo []string
		want    []string
	}{
		"no header": {
			from:    "sender@example.com",
			replyTo: nil,
			want:    []string{"sender@example.com"},
		},
		"header set": {
			from:    "sender@example.com",
			replyTo: []string{"reply-to@example.com"},
			want:    []string{"reply-to@example.com"},
		},
	}

	config := &config.ParsedConfig{}

	originalRecipientRaw := "private@example.com"
	originalRecipient, err := mail.ParseAddress(originalRecipientRaw)
	if err != nil {
		t.Fatalf("Invalid address: %v, %v", originalRecipientRaw, err)
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			message := toRFC5322LineDelimiter(`Foo: bar
Content-Type: text/plain

Message body
`)

			mailMessage, err := mail.ReadMessage(strings.NewReader(message))
			if err != nil {
				t.Fatal(err)
			}

			// Set headers
			mailMessage.Header[FromKey] = []string{tc.from}
			if tc.replyTo != nil {
				mailMessage.Header[ReplyToKey] = tc.replyTo
			}

			// Act
			if err := ProcessMessageHeader(config, mailMessage.Header, originalRecipient); err != nil {
				t.Fatal(err)
			}

			assertHeader(t, mailMessage.Header, ReplyToKey, tc.want)
		})
	}
}

func TestProcessMessageHeaderSubject(t *testing.T) {
	tests := map[string]struct {
		config  *config.ParsedConfig
		subject []string
		want    []string
	}{
		"without subject prefix": {
			config: &config.ParsedConfig{
				RawConfig: config.RawConfig{
					SubjectPrefix: "",
				},
			},
			subject: []string{"Test subject"},
			want:    []string{"Test subject"},
		},
		"with subject prefix": {
			config: &config.ParsedConfig{
				RawConfig: config.RawConfig{
					SubjectPrefix: "FORWARDER: ",
				},
			},
			subject: []string{"Test subject"},
			want:    []string{"FORWARDER: Test subject"},
		},
	}

	originalRecipientRaw := "public@example.com"
	originalRecipient, err := mail.ParseAddress(originalRecipientRaw)
	if err != nil {
		t.Fatalf("Invalid address: %v, %v", originalRecipientRaw, err)
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			message := toRFC5322LineDelimiter(`Foo: bar
Content-Type: text/plain

Message body
`)

			mailMessage, err := mail.ReadMessage(strings.NewReader(message))
			if err != nil {
				t.Fatal(err)
			}

			// Prepare headers test setup
			if len(tc.subject) > 0 {
				mailMessage.Header[SubjectKey] = tc.subject
			}

			// Act
			if err := ProcessMessageHeader(tc.config, mailMessage.Header, originalRecipient); err != nil {
				t.Fatal(err)
			}

			assertHeader(t, mailMessage.Header, SubjectKey, tc.want)
		})
	}
}

func TestProcessMessageHeaderNoDkimSignature(t *testing.T) {
	originalRecipientRaw := "public@example.com"
	originalRecipient, err := mail.ParseAddress(originalRecipientRaw)
	if err != nil {
		t.Fatalf("Invalid address: %v, %v", originalRecipientRaw, err)
	}

	message := toRFC5322LineDelimiter(`Foo: bar
X-SES-DKIM-SIGNATURE: a=rsa-sha256; q=dns/txt; b=f2pIQmy3y57b8pPsSDb9MdyHHBDYLn1yj6aBpCV1N8cCy2kva5d9S+5eAKMUeR8eDRRIJQzOcPIR9JflG8Gx1tKJrFSMCn4FFvowCpKqiGuwXJY+5I3A9T0uMY/JVuj+IXMwCYuI/6M0Fould0/D3AJDwZjNlVYkZUHnkP7zOlo=; c=relaxed/simple; s=ihchhvubuqgjsxyuhssfvqohv7z3u4hn; d=amazonses.com; t=1669144561; v=1; bh=VHKiB6Z/6y5PcMaJVbY5x4A0kvlzl8ASM0ker1qbjxY=; h=From:To:Cc:Bcc:Subject:Date:Message-ID:MIME-Version:Content-Type:X-SES-RECEIPT;
DKIM-Signature: v=1; a=rsa-sha256; q=dns/txt; c=relaxed/simple;
	s=o7bai5im5otrg6zk6gyjdrmo53gkx2er; d=excited-emu.awsapps.com;
	t=1669144560;
	h=Subject:From:To:Cc:Date:Mime-Version:Content-Type:References:Message-Id;
	bh=VHKiB6Z/6y5PcMaJVbY5x4A0kvlzl8ASM0ker1qbjxY=;
	b=uu/yvrdoee2/dbB3ZKRVKQ+65cuJGQkN4bbQ0HGhzhEQSXMm/MXEuKybMcUxrq2U
	lWkzi+CGh0K6AFGf6zRnZaHjkZ2w7D+inuxOz3Lnt8BkJeQfLUERP9NrgCSIZBTr9lo
	g2G8JW53QefUxD33abqp6yDL+jzBsCTVD7MThcwxwqa0QF8x9DkLkjbnzQkZ6BCrkAu
	W/Smg07IwhJsKgwjd6M6DqJ7R77hSeEnIA20Ulrd1+KlOnzwgBvDM4FRgiH8+UsFDVm
	kAqcWWY4142ptUK15n7R1IJHmnrjslE9/AM49gtrqbuGi81Mce8R0PaON3kiqfz0KzF
	iXaQEy8FGA==
DKIM-Signature: v=1; a=rsa-sha256; q=dns/txt; c=relaxed/simple;
	s=ihchhvubuqgjsxyuhssfvqohv7z3u4hn; d=amazonses.com; t=1669144560;
	h=Subject:From:To:Cc:Date:Mime-Version:Content-Type:References:Message-Id:Feedback-ID;
	bh=VHKiB6Z/6y5PcMaJVbY5x4A0kvlzl8ASM0ker1qbjxY=;
	b=mDtS3zTGYh5csNN3KfWskpZuwldq4ZIRhI6kAswVOpXxEWkjxliNGVL/EGW03XPn
	AsUAAtEsX9+c50ZnwEFNCt8unOQXVoBog6XrYxhsaQC5eqmTWoKUY6/mAFQ7H5ThY7N
	bnLK30yTfJhZ42PkhwX5YrR5NgG0VNG023+5lC2c=
X-Google-DKIM-Signature: v=1; a=rsa-sha256; c=relaxed/relaxed;
	d=1e100.net; s=20210112;
	h=to:from:subject:message-id:feedback-id:reply-to:date:mime-version
		:dkim-signature:x-gm-message-state:from:to:cc:subject:date
		:message-id:reply-to;
	bh=5Pbgk+B4uo7Me6CnXl5R90b7m7iObpcZOSL6iItr66s=;
	b=4sWMDyotELNk9B5Udnp05Fps7uvvEdW544m6+pmZe4rxeOvpcA9au3d3D2T4quY1UG
		64SXluPxE7gkLMXPQufrp0rCBkCmqYZGqAUtUSQXKWqr0lf8lHiCtT4P8xrGZxT2JcZE
		DUaEAb+joPqCFrxOdnvXgY7sWK//z3U4tDnRcJjiylHPJVzBALPXJACoPEheOkOIKW8m
		7uRH+VpvbBmFClrmZt8pvZypIogbqVbPJX5ZQfvjt7FV3guy+I2bFal42rjWYOpOiM1z
		NnaynsBzBlwFhQplKGgzEzxNi62KAQMAn/ACfEsStK1CKpHhqMJ5fNIK4F16SURimBGb
		hdYQ==
Content-Type: text/plain

Message body
`)

	mailMessage, err := mail.ReadMessage(strings.NewReader(message))
	if err != nil {
		t.Fatal(err)
	}

	// Act
	if err := ProcessMessageHeader(&config.ParsedConfig{}, mailMessage.Header, originalRecipient); err != nil {
		t.Fatal(err)
	}

	// No *-Dkim-Signature headers
	for k := range mailMessage.Header {
		if strings.HasSuffix(k, "Dkim-Signature") {
			t.Errorf("found unexpected header %s", k)
		}
	}
}

func TestProcessMessageHeaderTestMail(t *testing.T) {
	config := config.ParsedConfig{}

	originalRecipientRaw := "to@excited-emu.awsapps.com"
	originalRecipient, err := mail.ParseAddress(originalRecipientRaw)
	if err != nil {
		t.Fatalf("Invalid address: %v, %v", originalRecipientRaw, err)
	}

	reader, err := os.Open("../testdata/test-mail-with-attachment.eml")
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	mailMessage, err := mail.ReadMessage(reader)
	if err != nil {
		t.Fatal(err)
	}

	if err := ProcessMessageHeader(&config, mailMessage.Header, originalRecipient); err != nil {
		t.Fatal(err)
	}

	// Should be rewritten to the original recipient
	assertHeader(t, mailMessage.Header, FromKey, []string{"<to@excited-emu.awsapps.com>"})
	// Should be left unchanged
	assertHeader(t, mailMessage.Header, ToKey, []string{"=?UTF-8?Q?To?= <to@excited-emu.awsapps.com>, =?UTF-8?Q?Donald_Duck?= <donald.duck@excited-emu.awsapps.com>"})
	// Should be left unchanged
	assertHeader(t, mailMessage.Header, CcKey, []string{"=?UTF-8?Q?CC?= <cc@excited-emu.awsapps.com>, =?UTF-8?Q?Dagobert_Duck?= <dagobert.duck@excited-emu.awsapps.com>"})
	// Should be set to original sender
	assertHeader(t, mailMessage.Header, ReplyToKey, []string{"=?UTF-8?Q?Sender?= <sender@excited-emu.awsapps.com>"})
	// Should be left unchanged
	assertHeader(t, mailMessage.Header, SubjectKey, []string{"Test mail with attachment"})
	// Should not be present
	assertHeaderNotPresent(t, mailMessage.Header, ReturnPathKey)
	// Should not be present
	assertHeaderNotPresent(t, mailMessage.Header, SenderKey)
	// Should not be present
	assertHeaderNotPresent(t, mailMessage.Header, MessageIdKey)

	// No *-Dkim-Signature headers
	for k := range mailMessage.Header {
		if strings.HasSuffix(k, "Dkim-Signature") {
			t.Errorf("found unexpected header %s", k)
		}
	}
}

func assertHeader(t *testing.T, header mail.Header, headerKey string, want []string) {
	got := header[headerKey]
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("%s header (-want +got):\n%s", headerKey, diff)
	}
}

func assertHeaderNotPresent(t *testing.T, header mail.Header, headerKey string) {
	if _, found := header[headerKey]; found {
		t.Errorf("found unexpected header %s", headerKey)
	}
}
