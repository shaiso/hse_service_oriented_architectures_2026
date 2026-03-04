package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shaiso/marketplace/internal/model"
)

type PromoCodeRepo struct {
	pool *pgxpool.Pool
}

func NewPromoCodeRepo(pool *pgxpool.Pool) *PromoCodeRepo {
	return &PromoCodeRepo{pool: pool}
}

func (r *PromoCodeRepo) Create(ctx context.Context, promo *model.PromoCode) error {
	query := `
		INSERT INTO promo_codes (code, discount_type, discount_value, min_order_amount, max_uses, valid_from, valid_until)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, current_uses, active`

	err := r.pool.QueryRow(ctx, query,
		promo.Code, promo.DiscountType, promo.DiscountValue,
		promo.MinOrderAmount, promo.MaxUses, promo.ValidFrom, promo.ValidUntil,
	).Scan(&promo.ID, &promo.CurrentUses, &promo.Active)
	if err != nil {
		return fmt.Errorf("insert promo code: %w", err)
	}
	return nil
}

func (r *PromoCodeRepo) GetByCode(ctx context.Context, db DBTX, code string) (*model.PromoCode, error) {
	query := `
		SELECT id, code, discount_type, discount_value, min_order_amount,
		       max_uses, current_uses, valid_from, valid_until, active
		FROM promo_codes WHERE code = $1`

	var p model.PromoCode
	err := db.QueryRow(ctx, query, code).Scan(
		&p.ID, &p.Code, &p.DiscountType, &p.DiscountValue, &p.MinOrderAmount,
		&p.MaxUses, &p.CurrentUses, &p.ValidFrom, &p.ValidUntil, &p.Active)
	if err != nil {
		return nil, fmt.Errorf("get promo code by code: %w", err)
	}
	return &p, nil
}

func (r *PromoCodeRepo) GetByID(ctx context.Context, db DBTX, promoID uuid.UUID) (*model.PromoCode, error) {
	query := `
		SELECT id, code, discount_type, discount_value, min_order_amount,
		       max_uses, current_uses, valid_from, valid_until, active
		FROM promo_codes WHERE id = $1`

	var p model.PromoCode
	err := db.QueryRow(ctx, query, promoID).Scan(
		&p.ID, &p.Code, &p.DiscountType, &p.DiscountValue, &p.MinOrderAmount,
		&p.MaxUses, &p.CurrentUses, &p.ValidFrom, &p.ValidUntil, &p.Active)
	if err != nil {
		return nil, fmt.Errorf("get promo code by id: %w", err)
	}
	return &p, nil
}

func (r *PromoCodeRepo) IncrementUses(ctx context.Context, db DBTX, promoID uuid.UUID) error {
	query := `UPDATE promo_codes SET current_uses = current_uses + 1 WHERE id = $1`
	_, err := db.Exec(ctx, query, promoID)
	if err != nil {
		return fmt.Errorf("increment promo uses: %w", err)
	}
	return nil
}

func (r *PromoCodeRepo) DecrementUses(ctx context.Context, db DBTX, promoID uuid.UUID) error {
	query := `UPDATE promo_codes SET current_uses = current_uses - 1 WHERE id = $1 AND current_uses > 0`
	_, err := db.Exec(ctx, query, promoID)
	if err != nil {
		return fmt.Errorf("decrement promo uses: %w", err)
	}
	return nil
}
