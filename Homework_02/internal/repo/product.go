package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shaiso/marketplace/internal/model"
)

type ProductRepo struct {
	pool *pgxpool.Pool
}

func NewProductRepo(pool *pgxpool.Pool) *ProductRepo { return &ProductRepo{pool: pool} }

func (p *ProductRepo) Create(ctx context.Context, product *model.Product) error {
	query := `
		INSERT INTO products (name, description, price, stock, category, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	err := p.pool.QueryRow(ctx, query,
		product.Name,
		product.Description,
		product.Price,
		product.Stock,
		product.Category,
		product.Status,
	).Scan(&product.ID, &product.CreatedAt, &product.UpdatedAt)

	if err != nil {
		return fmt.Errorf("insert product: %w", err)
	}
	return nil
}

func (p *ProductRepo) GetByID(ctx context.Context, productID uuid.UUID) (*model.Product, error) {
	query := `
		SELECT id, name, description, price, stock, category, status, created_at, updated_at 
		FROM products WHERE id = $1`

	var product model.Product
	err := p.pool.QueryRow(ctx, query, productID).Scan(&product.ID, &product.Name, &product.Description,
		&product.Price, &product.Stock, &product.Category, &product.Status, &product.CreatedAt, &product.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("get product by id: %w", err)
	}
	return &product, nil
}

func (p *ProductRepo) GetAll(ctx context.Context, page, size int, status *string, category *string) ([]model.Product, int, error) {

	query := `
		SELECT id, name, description, price, stock, category, status, created_at, updated_at
		FROM products WHERE 1=1`

	countQuery := `
         SELECT COUNT(*) FROM products WHERE 1=1`

	args := []any{}
	argIdx := 1

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		countQuery += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *status)
		argIdx++
	}

	if category != nil {
		query += fmt.Sprintf(" AND category = $%d", argIdx)
		countQuery += fmt.Sprintf(" AND category = $%d", argIdx)
		args = append(args, *category)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	paginationArgs := append(args, size, page*size)

	var totalElements int
	err := p.pool.QueryRow(ctx, countQuery, args...).Scan(&totalElements)
	if err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	rows, err := p.pool.Query(ctx, query, paginationArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("get products: %w", err)
	}
	defer rows.Close()

	var products []model.Product
	for rows.Next() {
		var product model.Product
		err := rows.Scan(&product.ID, &product.Name, &product.Description,
			&product.Price, &product.Stock, &product.Category,
			&product.Status, &product.CreatedAt, &product.UpdatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, product)
	}

	return products, totalElements, nil
}

func (p *ProductRepo) Update(ctx context.Context, product *model.Product) error {
	query := `
		UPDATE products
		SET name = $2, description = $3, price = $4, stock = $5, category = $6, status = $7
		WHERE id = $1
		RETURNING updated_at`

	err := p.pool.QueryRow(ctx, query, product.ID, product.Name, product.Description,
		product.Price, product.Stock, product.Category, product.Status).Scan(&product.UpdatedAt)

	if err != nil {
		return fmt.Errorf("update product: %w", err)
	}

	return nil
}

func (p *ProductRepo) Delete(ctx context.Context, productID uuid.UUID) (*model.Product, error) {
	query := `
          UPDATE products SET status = 'ARCHIVED'
          WHERE id = $1
          RETURNING id, name, description, price, stock, category, status, created_at, updated_at`

	var product model.Product

	err := p.pool.QueryRow(ctx, query, productID).Scan(&product.ID, &product.Name, &product.Description, &product.Price,
		&product.Stock, &product.Category, &product.Status, &product.CreatedAt, &product.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("delete product: %w", err)
	}

	return &product, nil
}

func (p *ProductRepo) GetByIDWithDB(ctx context.Context, db DBTX, productID uuid.UUID) (*model.Product, error) {
	query := `
		SELECT id, name, description, price, stock, category, status, created_at, updated_at
		FROM products WHERE id = $1`

	var product model.Product
	err := db.QueryRow(ctx, query, productID).Scan(&product.ID, &product.Name, &product.Description,
		&product.Price, &product.Stock, &product.Category, &product.Status, &product.CreatedAt, &product.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get product by id: %w", err)
	}
	return &product, nil
}

func (p *ProductRepo) UpdateStock(ctx context.Context, db DBTX, productID uuid.UUID, delta int) error {
	query := `UPDATE products SET stock = stock + $2 WHERE id = $1`
	_, err := db.Exec(ctx, query, productID, delta)
	if err != nil {
		return fmt.Errorf("update stock: %w", err)
	}
	return nil
}

var (
	ErrNotFound = errors.New("not found")
)
