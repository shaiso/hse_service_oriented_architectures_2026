package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shaiso/marketplace/internal/model"
)

type UserOperationRepo struct {
	pool *pgxpool.Pool
}

func NewUserOperationRepo(pool *pgxpool.Pool) *UserOperationRepo {
	return &UserOperationRepo{pool: pool}
}

func (r *UserOperationRepo) Create(ctx context.Context, db DBTX, op *model.UserOperation) error {
	query := `
		INSERT INTO user_operations (user_id, operation_type)
		VALUES ($1, $2)
		RETURNING id, created_at`

	err := db.QueryRow(ctx, query, op.UserID, op.OperationType).Scan(&op.ID, &op.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert user operation: %w", err)
	}
	return nil
}

func (r *UserOperationRepo) GetLastByType(ctx context.Context, userID uuid.UUID, opType model.OperationType) (*model.UserOperation, error) {
	query := `
		SELECT id, user_id, operation_type, created_at
		FROM user_operations
		WHERE user_id = $1 AND operation_type = $2
		ORDER BY created_at DESC
		LIMIT 1`

	var op model.UserOperation
	err := r.pool.QueryRow(ctx, query, userID, opType).Scan(
		&op.ID, &op.UserID, &op.OperationType, &op.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get last user operation: %w", err)
	}
	return &op, nil
}
