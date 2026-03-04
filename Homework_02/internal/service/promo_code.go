package service

import (
	"context"
	"fmt"

	"github.com/shaiso/marketplace/generated"
	"github.com/shaiso/marketplace/internal/model"
	"github.com/shaiso/marketplace/internal/repo"
)

type PromoCodeService struct {
	repo *repo.PromoCodeRepo
}

func NewPromoCodeService(repo *repo.PromoCodeRepo) *PromoCodeService {
	return &PromoCodeService{repo: repo}
}

func (s *PromoCodeService) Create(ctx context.Context, req generated.PromoCodeCreate) (*generated.PromoCodeResponse, error) {
	if err := validatePromoCodeCreate(req); err != nil {
		return nil, err
	}

	promo := model.PromoCode{
		Code:           req.Code,
		DiscountType:   model.DiscountType(req.DiscountType),
		DiscountValue:  req.DiscountValue,
		MinOrderAmount: req.MinOrderAmount,
		MaxUses:        req.MaxUses,
		ValidFrom:      req.ValidFrom,
		ValidUntil:     req.ValidUntil,
	}

	if err := s.repo.Create(ctx, &promo); err != nil {
		return nil, fmt.Errorf("create promo code: %w", err)
	}

	return toPromoCodeResponse(&promo), nil
}

func validatePromoCodeCreate(req generated.PromoCodeCreate) error {
	fields := make(map[string]string)

	if len(req.Code) < 4 || len(req.Code) > 20 {
		fields["code"] = "must be between 4 and 20 characters"
	}

	if req.DiscountValue < 0.01 {
		fields["discount_value"] = "must be greater than 0"
	}

	if req.MinOrderAmount < 0 {
		fields["min_order_amount"] = "must be >= 0"
	}

	if req.MaxUses < 1 {
		fields["max_uses"] = "must be >= 1"
	}

	if !req.ValidUntil.After(req.ValidFrom) {
		fields["valid_until"] = "must be after valid_from"
	}

	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

func toPromoCodeResponse(p *model.PromoCode) *generated.PromoCodeResponse {
	return &generated.PromoCodeResponse{
		Id:             p.ID,
		Code:           p.Code,
		DiscountType:   generated.DiscountType(p.DiscountType),
		DiscountValue:  p.DiscountValue,
		MinOrderAmount: p.MinOrderAmount,
		MaxUses:        p.MaxUses,
		CurrentUses:    p.CurrentUses,
		ValidFrom:      p.ValidFrom,
		ValidUntil:     p.ValidUntil,
		Active:         p.Active,
	}
}
