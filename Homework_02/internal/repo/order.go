package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shaiso/marketplace/internal/model"
)

type OrderRepo struct {
	pool *pgxpool.Pool
}

func NewOrderRepo(pool *pgxpool.Pool) *OrderRepo {
	return &OrderRepo{pool: pool}
}

func (o *OrderRepo) Pool() *pgxpool.Pool {
	return o.pool
}

func (o *OrderRepo) Create(ctx context.Context, db DBTX, order *model.Order, items []model.OrderItem) error {
	orderQuery := `
		INSERT INTO orders (user_id, status, promo_code_id, total_amount, discount_amount)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	err := db.QueryRow(ctx, orderQuery,
		order.UserID, order.Status, order.PromoCodeID,
		order.TotalAmount, order.DiscountAmount).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	return o.insertItems(ctx, db, order.ID, items)
}

// GetByID получает заказ по ID.
func (o *OrderRepo) GetByID(ctx context.Context, orderID uuid.UUID) (*model.Order, error) {
	return o.getByID(ctx, o.pool, orderID)
}

// GetByIDWithDB получает заказ по ID, используя переданное соединение.
func (o *OrderRepo) GetByIDWithDB(ctx context.Context, db DBTX, orderID uuid.UUID) (*model.Order, error) {
	return o.getByID(ctx, db, orderID)
}

func (o *OrderRepo) getByID(ctx context.Context, db DBTX, orderID uuid.UUID) (*model.Order, error) {
	query := `
		SELECT id, user_id, status, promo_code_id, total_amount, discount_amount, created_at, updated_at
		FROM orders WHERE id = $1`

	var order model.Order
	err := db.QueryRow(ctx, query, orderID).Scan(
		&order.ID, &order.UserID, &order.Status, &order.PromoCodeID,
		&order.TotalAmount, &order.DiscountAmount, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get order by id: %w", err)
	}
	return &order, nil
}

// GetItemsByOrderID получает позиции заказа.
func (o *OrderRepo) GetItemsByOrderID(ctx context.Context, orderID uuid.UUID) ([]model.OrderItem, error) {
	return o.getItems(ctx, o.pool, orderID)
}

// GetItemsByOrderIDWithDB — то же, но с переданным соединением.
func (o *OrderRepo) GetItemsByOrderIDWithDB(ctx context.Context, db DBTX, orderID uuid.UUID) ([]model.OrderItem, error) {
	return o.getItems(ctx, db, orderID)
}

func (o *OrderRepo) getItems(ctx context.Context, db DBTX, orderID uuid.UUID) ([]model.OrderItem, error) {
	query := `
		SELECT id, order_id, product_id, quantity, price_at_order
		FROM order_items WHERE order_id = $1`

	rows, err := db.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("get order items: %w", err)
	}
	defer rows.Close()

	var items []model.OrderItem
	for rows.Next() {
		var item model.OrderItem
		err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.PriceAtOrder)
		if err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

// GetActiveOrderByUserID проверяет, есть ли у пользователя активный заказ (CREATED или PAYMENT_PENDING).
func (o *OrderRepo) GetActiveOrderByUserID(ctx context.Context, userID uuid.UUID) (*model.Order, error) {
	query := `
		SELECT id, user_id, status, promo_code_id, total_amount, discount_amount, created_at, updated_at
		FROM orders
		WHERE user_id = $1 AND status IN ('CREATED', 'PAYMENT_PENDING')
		LIMIT 1`

	var order model.Order
	err := o.pool.QueryRow(ctx, query, userID).Scan(
		&order.ID, &order.UserID, &order.Status, &order.PromoCodeID,
		&order.TotalAmount, &order.DiscountAmount, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get active order: %w", err)
	}
	return &order, nil
}

// UpdateStatus изменяет статус заказа.
func (o *OrderRepo) UpdateStatus(ctx context.Context, db DBTX, orderID uuid.UUID, status model.OrderStatus) error {
	query := `UPDATE orders SET status = $2 WHERE id = $1`
	_, err := db.Exec(ctx, query, orderID, status)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	return nil
}

// UpdateOrder обновляет суммы заказа и заменяет позиции.
func (o *OrderRepo) UpdateOrder(ctx context.Context, db DBTX, order *model.Order, items []model.OrderItem) error {
	updateQuery := `
		UPDATE orders SET total_amount = $2, discount_amount = $3, promo_code_id = $4
		WHERE id = $1
		RETURNING updated_at`

	err := db.QueryRow(ctx, updateQuery,
		order.ID, order.TotalAmount, order.DiscountAmount, order.PromoCodeID).Scan(&order.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	_, err = db.Exec(ctx, `DELETE FROM order_items WHERE order_id = $1`, order.ID)
	if err != nil {
		return fmt.Errorf("delete old items: %w", err)
	}

	return o.insertItems(ctx, db, order.ID, items)
}

func (o *OrderRepo) insertItems(ctx context.Context, db DBTX, orderID uuid.UUID, items []model.OrderItem) error {
	itemQuery := `
		INSERT INTO order_items (order_id, product_id, quantity, price_at_order)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	for i := range items {
		items[i].OrderID = orderID
		err := db.QueryRow(ctx, itemQuery,
			items[i].OrderID, items[i].ProductID, items[i].Quantity, items[i].PriceAtOrder).Scan(&items[i].ID)
		if err != nil {
			return fmt.Errorf("insert order item: %w", err)
		}
	}
	return nil
}
