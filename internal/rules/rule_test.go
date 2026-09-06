package rules

import (
	"strings"
	"testing"
)

func TestCategories(t *testing.T) {
	set, err := New()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		line string
		want string
	}{
		{name: "AWS access", line: "AKIA" + strings.Repeat("A", 16), want: "aws-access-key"},
		{name: "AWS temporary", line: "ASIA" + strings.Repeat("B", 16), want: "aws-access-key"},
		{name: "AWS secret", line: "aws_secret_access_key = " + strings.Repeat("A", 40), want: "aws-secret-key"},
		{name: "AWS secret punctuation", line: "aws_secret_access_key = " + strings.Repeat("A", 39) + "/", want: "aws-secret-key"},
		{name: "GCP context", line: `"type": "service_account"`, want: "gcp-service-account"},
		{name: "GitHub PAT", line: "ghp_" + strings.Repeat("a", 36), want: "github-token"},
		{name: "GitHub OAuth", line: "gho_" + strings.Repeat("a", 36), want: "github-token"},
		{name: "Slack", line: "xoxb-0000000000-synthetic", want: "slack-token"},
		{name: "Stripe live", line: "sk_live_" + strings.Repeat("a", 24), want: "stripe-key"},
		{name: "Stripe test", line: "sk_test_" + strings.Repeat("a", 24), want: "stripe-key"},
		{name: "JWT", line: "eyJ" + strings.Repeat("a", 8) + ".aaaaaaaa.bbbbbbbb", want: "jwt"},
		{name: "PEM", line: "-----BEGIN PRIVATE KEY-----", want: "pem-private-key"},
		{name: "RSA PEM", line: "-----BEGIN RSA PRIVATE KEY-----", want: "pem-private-key"},
		{name: "OpenSSH PEM", line: "-----BEGIN OPENSSH PRIVATE KEY-----", want: "pem-private-key"},
		{name: "generic", line: `password = "synthetic-only"`, want: "generic-assignment"},
		{name: "API assignment", line: `"api_key": "synthetic-only"`, want: "generic-assignment"},
		{name: "Postgres", line: "postgres://demo:synthetic@localhost/db", want: "database-url"},
		{name: "MySQL", line: "mysql://demo:synthetic@localhost/db", want: "database-url"},
		{name: "MongoDB", line: "mongodb://demo:synthetic@localhost/db", want: "database-url"},
		{name: "MongoDB SRV", line: "mongodb+srv://demo:synthetic@localhost/db", want: "database-url"},
		{name: "iyzico", line: `iyzico_api_key = "synthetic-only"`, want: "iyzico"},
		{name: "PayTR", line: `PAYTR_MERCHANT_SALT=synthetic-only`, want: "paytr"},
		{name: "Param", line: `param_client_password=synthetic-only`, want: "param"},
		{name: "Netgsm", line: `netgsm_password=synthetic-only`, want: "netgsm"},
		{name: "bank", line: `banka_api_key=synthetic-only`, want: "turkish-bank-api"},
		{name: "short token", line: "ghp_short"},
		{name: "public key", line: "-----BEGIN PUBLIC KEY-----"},
		{name: "passwordless DB", line: "postgres://localhost/demo"},
		{name: "environment reference", line: `password = "${DB_PASSWORD}"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := set.Find(tt.line)
			if tt.want == "" {
				if len(matches) != 0 {
					t.Fatalf("unexpected matches: %+v", matches)
				}
				return
			}
			if len(matches) != 1 || matches[0].Rule != tt.want {
				t.Fatalf("matches = %+v, want one %s", matches, tt.want)
			}
			if matches[0].Start < 0 || matches[0].End > len(tt.line) || matches[0].Start >= matches[0].End {
				t.Fatal("invalid match span")
			}
		})
	}
}

func TestFindOwnershipAndMultipleMatches(t *testing.T) {
	set, err := New()
	if err != nil {
		t.Fatal(err)
	}
	line := "ghp_" + strings.Repeat("a", 36) + " ghp_" + strings.Repeat("b", 36)
	matches := set.Find(line)
	if len(matches) != 2 {
		t.Fatalf("got %d matches", len(matches))
	}
	matches[0].Rule = "changed"
	if set.Find(line)[0].Rule != "github-token" {
		t.Fatal("caller mutated rule storage")
	}
}
