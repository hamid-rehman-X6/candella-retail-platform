package register

import (
	"context"
	"errors"
	"testing"
)

func TestClosingFigures(t *testing.T) {
	cases := []struct {
		name               string
		opening, cashSales int64
		in                 CloseInput
		expected, variance int64
		err                error
	}{
		{"balanced drawer needs no reason", 5000, 12000, CloseInput{CountedCashMinor: 17000, ExpectedCashMinor: 17000}, 17000, 0, nil},
		{"short drawer with a reason", 5000, 12000, CloseInput{CountedCashMinor: 16500, ExpectedCashMinor: 17000, VarianceReason: "Gave extra change"}, 17000, -500, nil},
		{"over drawer with a reason", 0, 1000, CloseInput{CountedCashMinor: 1200, ExpectedCashMinor: 1000, VarianceReason: "Tip left in drawer"}, 1000, 200, nil},
		{"short drawer without a reason", 5000, 12000, CloseInput{CountedCashMinor: 16500, ExpectedCashMinor: 17000}, 0, 0, ErrReasonRequired},
		{"reason too short", 5000, 0, CloseInput{CountedCashMinor: 4000, ExpectedCashMinor: 5000, VarianceReason: "ok"}, 0, 0, ErrReasonRequired},
		{"a sale landed after the closer looked", 5000, 12000, CloseInput{CountedCashMinor: 15000, ExpectedCashMinor: 15000}, 0, 0, ErrExpectedChanged},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expected, variance, err := closingFigures(tc.opening, tc.cashSales, tc.in)
			if !errors.Is(err, tc.err) || expected != tc.expected || variance != tc.variance {
				t.Fatalf("got expected=%d variance=%d err=%v; want %d, %d, %v",
					expected, variance, err, tc.expected, tc.variance, tc.err)
			}
		})
	}
}

// fakeRepo records whether the service let a call through.
type fakeRepo struct{ calls int }

func (f *fakeRepo) ListRegisters(context.Context, string, string) ([]Register, error) {
	f.calls++
	return nil, nil
}
func (f *fakeRepo) CreateRegister(_ context.Context, _ string, in CreateRegisterInput) (*Register, error) {
	f.calls++
	return &Register{Name: in.Name}, nil
}
func (f *fakeRepo) Open(context.Context, string, Actor, OpenInput) (*Shift, error) {
	f.calls++
	return &Shift{}, nil
}
func (f *fakeRepo) Get(context.Context, string, Actor, string) (*Shift, error) {
	f.calls++
	return &Shift{}, nil
}
func (f *fakeRepo) Close(context.Context, string, Actor, string, CloseInput) (*Shift, error) {
	f.calls++
	return &Shift{}, nil
}
func (f *fakeRepo) List(_ context.Context, _ string, _ Actor, filter ShiftFilter) ([]Shift, bool, error) {
	f.calls++
	return nil, false, nil
}

func TestServiceValidation(t *testing.T) {
	ctx := context.Background()
	cashier := Actor{UserID: "usr_aaaaaaaaaaaa", Name: "Ayesha", Role: "member"}
	owner := Actor{UserID: "usr_bbbbbbbbbbbb", Name: "Hamid", Role: "owner"}
	const registerID, shiftID = "reg_aaaaaaaaaaaa", "shift_aaaaaaaaaaaa"

	cases := []struct {
		name string
		call func(*Service) error
		want error
	}{
		{"open without a counter", func(s *Service) error {
			_, err := s.Open(ctx, "t", cashier, OpenInput{StoreID: "s", OpeningCashMinor: 100})
			return err
		}, ErrRegisterRequired},
		{"open on a malformed counter", func(s *Service) error {
			_, err := s.Open(ctx, "t", cashier, OpenInput{StoreID: "s", RegisterID: "reg_x", OpeningCashMinor: 100})
			return err
		}, ErrRegisterNotFound},
		{"negative opening float", func(s *Service) error {
			_, err := s.Open(ctx, "t", cashier, OpenInput{StoreID: "s", RegisterID: registerID, OpeningCashMinor: -1})
			return err
		}, ErrInvalid},
		{"opening float too large", func(s *Service) error {
			_, err := s.Open(ctx, "t", cashier, OpenInput{StoreID: "s", RegisterID: registerID, OpeningCashMinor: maxCashMinor + 1})
			return err
		}, ErrInvalid},
		{"open with zero float", func(s *Service) error {
			_, err := s.Open(ctx, "t", cashier, OpenInput{StoreID: "s", RegisterID: registerID})
			return err
		}, nil},
		{"negative counted cash", func(s *Service) error {
			_, err := s.Close(ctx, "t", cashier, shiftID, CloseInput{CountedCashMinor: -1})
			return err
		}, ErrInvalid},
		{"reason over 500 characters", func(s *Service) error {
			long := make([]byte, 501)
			for i := range long {
				long[i] = 'x'
			}
			_, err := s.Close(ctx, "t", cashier, shiftID, CloseInput{VarianceReason: string(long)})
			return err
		}, ErrInvalid},
		{"close a malformed shift", func(s *Service) error {
			_, err := s.Close(ctx, "t", cashier, "ord_aaaaaaaaaaaa", CloseInput{})
			return err
		}, ErrNotFound},
		{"cashier cannot add counters", func(s *Service) error {
			_, err := s.CreateRegister(ctx, "t", cashier, CreateRegisterInput{StoreID: "s", Name: "Counter 2"})
			return err
		}, ErrForbidden},
		{"blank counter name", func(s *Service) error {
			_, err := s.CreateRegister(ctx, "t", owner, CreateRegisterInput{StoreID: "s", Name: "   "})
			return err
		}, ErrInvalidName},
		{"owner adds a counter", func(s *Service) error {
			_, err := s.CreateRegister(ctx, "t", owner, CreateRegisterInput{StoreID: "s", Name: " Counter 2 "})
			return err
		}, nil},
		{"unknown status filter", func(s *Service) error {
			_, _, err := s.List(ctx, "t", owner, ShiftFilter{StoreID: "s", Status: "pending"})
			return err
		}, ErrInvalidFilter},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepo{}
			err := tc.call(NewService(repo))
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if tc.want != nil && repo.calls != 0 {
				t.Fatal("an invalid request reached the repository")
			}
			if tc.want == nil && repo.calls != 1 {
				t.Fatal("a valid request did not reach the repository")
			}
		})
	}
}
