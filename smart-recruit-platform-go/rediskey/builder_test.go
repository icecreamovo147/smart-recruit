package rediskey

import "testing"

func TestBuilderRequiresServicePrefix(t *testing.T) {
	if _, err := NewBuilder(""); err == nil {
		t.Fatal("expected blank service prefix to fail")
	}
}

func TestBuilderPrefixesKeysAndChannels(t *testing.T) {
	builder := MustBuilder("identity")

	if got := builder.Key("token_version", "42"); got != "identity:token_version:42" {
		t.Fatalf("unexpected key: %q", got)
	}
	if got := builder.Channel("notification", "staff", "42"); got != "identity:notification:staff:42" {
		t.Fatalf("unexpected channel: %q", got)
	}
	if !builder.HasPrefix("identity:token_version:42") {
		t.Fatal("expected key to match builder prefix")
	}
}

func TestBuilderRejectsBlankParts(t *testing.T) {
	builder := MustBuilder("gateway")
	defer func() {
		if recover() == nil {
			t.Fatal("expected blank key part to panic")
		}
	}()
	_ = builder.Key("quota", "")
}

func TestBuilderSupportsCustomSeparator(t *testing.T) {
	builder := MustBuilder("worker", WithSeparator("|"))
	if got := builder.Key("inbox", "consumer"); got != "worker|inbox|consumer" {
		t.Fatalf("unexpected custom separator key: %q", got)
	}
}
