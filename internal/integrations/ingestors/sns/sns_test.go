package sns

import "testing"

func TestHostPattern(t *testing.T) {
	for host, want := range map[string]bool{
		"sns.us-east-1.amazonaws.com":              true,
		"sns.us-gov-west-1.amazonaws.com":          true,
		"sns.cn-north-1.amazonaws.com.cn":          true,
		"sns.s3-external-1.amazonaws.com":          false,
		"sns.s3-accelerate.amazonaws.com":          false,
		"sns.us-east-1.amazonaws.com.example.test": false,
	} {
		if got := hostPattern.MatchString(host); got != want {
			t.Errorf("hostPattern.MatchString(%q) = %v, want %v", host, got, want)
		}
	}
}

func TestBuildSignatureRejectsNewlineOutsideMessage(t *testing.T) {
	if _, err := (&Payload{Type: "Notification\nTopicArn\nother", Message: "a\nb"}).BuildSignature(); err == nil {
		t.Fatal("expected an error")
	}
}
