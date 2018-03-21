package mailer

import "testing"

func TestToStatusInFragment(t *testing.T) {
	str := toStatusInFragment([]RecipientStatus{
		RecipientStatuses.Get("new"),
		RecipientStatuses.Get("unsubscribing"),
	})

	if expected := "status in ('new', 'unsubscribing')"; str != expected {
		t.Errorf("got %q, want %q", str, expected)
	}
}
