package message

import (
	"bytes"
	"io"
	"net/mail"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBuildMail(t *testing.T) {
	// RFC5322 uses CRLF!
	rawMail := []byte(strings.Replace(`From: sender@example.com
To: recipient@example.com
Subject: Test email (contains an attachment)
MIME-Version: 1.0
Content-type: Multipart/Mixed; boundary=\"NextPart\"


--NextPart
Content-Type: text/plain

This is the message body.

--NextPart
Content-Type: text/plain;
Content-Disposition: attachment; filename=\"attachment.txt\"

This is the text in the attachment.

--NextPart--
`, "\n", RFC5322LineDelimiter, -1))

	reader := bytes.NewReader(rawMail)
	mailMessage, err := mail.ReadMessage(reader)
	if err != nil {
		t.Fatal(err)
	}

	mailBody, err := io.ReadAll(mailMessage.Body)
	if err != nil {
		t.Fatal(err)
	}

	bufferedMessage := &BufferedMessage{
		Header: mailMessage.Header,
		Body:   mailBody,
	}

	mailData, err := BuildMail(bufferedMessage)
	if err != nil {
		t.Fatal(err)
	}
	got := string(mailData)

	bytes, err := os.ReadFile("testdata/mail-with-attachment.mail")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(string(bytes), "\n", RFC5322LineDelimiter, -1)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mail (-want +got):\n%s", diff)
	}
}
