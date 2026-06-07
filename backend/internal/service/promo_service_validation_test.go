//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/socialops/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type promoValidationRepoStub struct {
	code        *PromoCode
	createCalls int
	updateCalls int
	deleteCalls int
}

func (r *promoValidationRepoStub) Create(_ context.Context, code *PromoCode) error {
	r.createCalls++
	copy := *code
	if copy.ID == 0 {
		copy.ID = 1
	}
	r.code = &copy
	return nil
}

func (r *promoValidationRepoStub) GetByID(_ context.Context, id int64) (*PromoCode, error) {
	if r.code == nil || r.code.ID != id {
		return nil, ErrPromoCodeNotFound
	}
	copy := *r.code
	return &copy, nil
}

func (r *promoValidationRepoStub) GetByCode(context.Context, string) (*PromoCode, error) {
	panic("unexpected GetByCode call")
}

func (r *promoValidationRepoStub) GetByCodeForUpdate(context.Context, string) (*PromoCode, error) {
	panic("unexpected GetByCodeForUpdate call")
}

func (r *promoValidationRepoStub) Update(_ context.Context, code *PromoCode) error {
	r.updateCalls++
	copy := *code
	copy.UpdatedAt = time.Now().UTC()
	r.code = &copy
	return nil
}

func (r *promoValidationRepoStub) Delete(_ context.Context, id int64) error {
	r.deleteCalls++
	if r.code == nil || r.code.ID != id {
		return ErrPromoCodeNotFound
	}
	r.code = nil
	return nil
}

func (r *promoValidationRepoStub) List(context.Context, pagination.PaginationParams) ([]PromoCode, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (r *promoValidationRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string) ([]PromoCode, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (r *promoValidationRepoStub) CreateUsage(context.Context, *PromoCodeUsage) error {
	panic("unexpected CreateUsage call")
}

func (r *promoValidationRepoStub) GetUsageByPromoCodeAndUser(context.Context, int64, int64) (*PromoCodeUsage, error) {
	panic("unexpected GetUsageByPromoCodeAndUser call")
}

func (r *promoValidationRepoStub) ListUsagesByPromoCode(context.Context, int64, pagination.PaginationParams) ([]PromoCodeUsage, *pagination.PaginationResult, error) {
	panic("unexpected ListUsagesByPromoCode call")
}

func (r *promoValidationRepoStub) IncrementUsedCount(context.Context, int64) error {
	panic("unexpected IncrementUsedCount call")
}

func TestPromoServiceCreateRejectsInvalidNumericInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input *CreatePromoCodeInput
	}{
		{
			name: "negative bonus amount",
			input: &CreatePromoCodeInput{
				Code:        "NEGATIVE",
				BonusAmount: -1,
				MaxUses:     0,
			},
		},
		{
			name: "negative max uses",
			input: &CreatePromoCodeInput{
				Code:        "NEGATIVEUSES",
				BonusAmount: 1,
				MaxUses:     -1,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &promoValidationRepoStub{}
			svc := NewPromoService(repo, nil, nil, nil, nil)

			code, err := svc.Create(context.Background(), tc.input)

			require.Nil(t, code)
			require.ErrorIs(t, err, ErrPromoCodeInvalid)
			require.Equal(t, 0, repo.createCalls)
		})
	}
}

func TestPromoServiceUpdateRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input *UpdatePromoCodeInput
	}{
		{
			name:  "blank code",
			input: &UpdatePromoCodeInput{Code: testPtrString("   ")},
		},
		{
			name:  "negative bonus amount",
			input: &UpdatePromoCodeInput{BonusAmount: testPtrFloat64(-0.01)},
		},
		{
			name:  "negative max uses",
			input: &UpdatePromoCodeInput{MaxUses: testPtrInt(-1)},
		},
		{
			name:  "invalid status",
			input: &UpdatePromoCodeInput{Status: testPtrString("archived")},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &promoValidationRepoStub{
				code: &PromoCode{
					ID:          7,
					Code:        "WELCOME",
					BonusAmount: 5,
					MaxUses:     10,
					UsedCount:   1,
					Status:      PromoCodeStatusActive,
					CreatedAt:   time.Now().UTC().Add(-time.Hour),
					UpdatedAt:   time.Now().UTC().Add(-time.Hour),
				},
			}
			svc := NewPromoService(repo, nil, nil, nil, nil)

			code, err := svc.Update(context.Background(), 7, tc.input)

			require.Nil(t, code)
			require.ErrorIs(t, err, ErrPromoCodeInvalid)
			require.Equal(t, 0, repo.updateCalls)
			require.Equal(t, "WELCOME", repo.code.Code)
			require.Equal(t, PromoCodeStatusActive, repo.code.Status)
		})
	}
}

func TestPromoServiceDeleteRejectsUsedPromoCode(t *testing.T) {
	t.Parallel()

	repo := &promoValidationRepoStub{
		code: &PromoCode{
			ID:          9,
			Code:        "USEDPROMO",
			BonusAmount: 5,
			MaxUses:     10,
			UsedCount:   1,
			Status:      PromoCodeStatusActive,
			CreatedAt:   time.Now().UTC().Add(-time.Hour),
			UpdatedAt:   time.Now().UTC().Add(-time.Hour),
		},
	}
	svc := NewPromoService(repo, nil, nil, nil, nil)

	err := svc.Delete(context.Background(), 9)

	require.ErrorIs(t, err, ErrPromoCodeAlreadyUsed)
	require.Equal(t, 0, repo.deleteCalls)
	require.NotNil(t, repo.code)
}
