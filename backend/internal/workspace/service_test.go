package workspace

import (
	"context"
	"testing"
)

// fakeRepo records the workspace handed to CreateWithOwner so tests can inspect
// the values the service chose.
type fakeRepo struct {
	created *Workspace
}

func (f *fakeRepo) SlugExists(context.Context, string) (bool, error) { return false, nil }

func (f *fakeRepo) CreateWithOwner(_ context.Context, w *Workspace, _ string) error {
	f.created = w
	return nil
}

func (f *fakeRepo) ListForUser(context.Context, string) ([]Workspace, error) { return nil, nil }

func (f *fakeRepo) GetMembership(context.Context, string, string) (*Membership, error) {
	return nil, ErrNoMembership
}

func TestCreateCurrency(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		want     string
	}{
		{"defaults to PKR when omitted", "", "PKR"},
		{"defaults to PKR when malformed", "RUPEES", "PKR"},
		{"normalises an explicit code", " usd ", "USD"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepo{}
			_, err := NewService(repo).Create(context.Background(), "usr_test", createRequest{
				Name:         "Karachi Mart",
				CurrencyCode: tc.currency,
			})
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			if repo.created.CurrencyCode != tc.want {
				t.Fatalf("currency = %q, want %q", repo.created.CurrencyCode, tc.want)
			}
		})
	}
}
