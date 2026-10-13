package catalog

import (
	"context"
	"testing"
)

type movementRepo struct {
	Repository
	called bool
	input  MovementInput
}

func (r *movementRepo) RecordMovement(_ context.Context, _, _ string, in MovementInput) (*Movement, error) {
	r.called = true
	r.input = in
	return &Movement{Kind: in.Kind}, nil
}

func ptr(n int64) *int64 { return &n }

func TestRecordMovementRejectsInvalidOperationsBeforeWriting(t *testing.T) {
	cases := []struct {
		name  string
		input MovementInput
	}{
		{"empty variant", MovementInput{Kind: "receive", Quantity: ptr(2)}},
		{"zero receipt", MovementInput{VariantID: "var_123456789012", Kind: "receive", Quantity: ptr(0)}},
		{"negative receipt", MovementInput{VariantID: "var_123456789012", Kind: "receive", Quantity: ptr(-1)}},
		{"ambiguous receipt", MovementInput{VariantID: "var_123456789012", Kind: "receive", Quantity: ptr(2), CountedQuantity: ptr(3)}},
		{"unexplained count", MovementInput{VariantID: "var_123456789012", Kind: "count", CountedQuantity: ptr(3)}},
		{"negative count", MovementInput{VariantID: "var_123456789012", Kind: "count", CountedQuantity: ptr(-1), Note: "Counted"}},
		{"unknown kind", MovementInput{VariantID: "var_123456789012", Kind: "sale", Quantity: ptr(1)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &movementRepo{}
			_, err := NewService(repo).RecordMovement(context.Background(), "ten_123456789012", "usr_123456789012", tc.input)
			if err != ErrInvalid {
				t.Fatalf("error = %v, want ErrInvalid", err)
			}
			if repo.called {
				t.Fatal("invalid movement reached the repository")
			}
		})
	}
}

func TestRecordMovementNormalizesValidReceipt(t *testing.T) {
	repo := &movementRepo{}
	movement, err := NewService(repo).RecordMovement(context.Background(), "ten_123456789012", "usr_123456789012", MovementInput{
		VariantID: " var_123456789012 ", Kind: "receive", Quantity: ptr(5), Note: " New delivery ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !repo.called || repo.input.VariantID != "var_123456789012" || repo.input.Note != "New delivery" || movement.Kind != "receive" {
		t.Fatalf("receipt was not forwarded correctly: %+v, %+v", repo.input, movement)
	}
}
