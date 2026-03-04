package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shaiso/marketplace/generated"
	"github.com/shaiso/marketplace/internal/model"
	"github.com/shaiso/marketplace/internal/repo"
)

type ProductService struct {
	repo *repo.ProductRepo
}

func NewProductService(repo *repo.ProductRepo) *ProductService {
	return &ProductService{repo: repo}
}

func (s *ProductService) Create(ctx context.Context, req generated.ProductCreate) (*generated.ProductResponse, error) {
	if err := validateProductCreate(req); err != nil {
		return nil, err
	}

	product := model.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		Category:    req.Category,
		Status:      model.ProductStatus(req.Status),
	}

	if err := s.repo.Create(ctx, &product); err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}

	return toProductResponse(&product), nil
}

func (s *ProductService) GetByID(ctx context.Context, id uuid.UUID) (*generated.ProductResponse, error) {
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrNotFound
		}
		return nil, fmt.Errorf("get product: %w", err)
	}

	return toProductResponse(product), nil
}

func (s *ProductService) List(ctx context.Context, params generated.GetProductsParams) (*generated.ProductListResponse, error) {
	page := 0
	size := 20
	if params.Page != nil {
		page = *params.Page
	}
	if params.Size != nil {
		size = *params.Size
	}

	var status *string
	if params.Status != nil {
		st := string(*params.Status)
		status = &st
	}

	products, total, err := s.repo.GetAll(ctx, page, size, status, params.Category)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}

	content := make([]generated.ProductResponse, len(products))
	for i := range products {
		content[i] = *toProductResponse(&products[i])
	}

	return &generated.ProductListResponse{
		Content:       content,
		TotalElements: total,
		Page:          page,
		Size:          size,
	}, nil
}

func (s *ProductService) Update(ctx context.Context, id uuid.UUID, req generated.ProductUpdate) (*generated.ProductResponse, error) {
	if err := validateProductUpdate(req); err != nil {
		return nil, err
	}

	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrNotFound
		}
		return nil, fmt.Errorf("get product for update: %w", err)
	}

	product.Name = req.Name
	product.Description = req.Description
	product.Price = req.Price
	product.Stock = req.Stock
	product.Category = req.Category
	product.Status = model.ProductStatus(req.Status)

	if err := s.repo.Update(ctx, product); err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}

	return toProductResponse(product), nil
}

func (s *ProductService) CreateWithSeller(ctx context.Context, req generated.ProductCreate, sellerID uuid.UUID) (*generated.ProductResponse, error) {
	if err := validateProductCreate(req); err != nil {
		return nil, err
	}

	product := model.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		Category:    req.Category,
		Status:      model.ProductStatus(req.Status),
		SellerID:    sellerID,
	}

	if err := s.repo.Create(ctx, &product); err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}

	return toProductResponse(&product), nil
}

func (s *ProductService) CheckOwnership(ctx context.Context, productID, sellerID uuid.UUID) error {
	product, err := s.repo.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repo.ErrNotFound
		}
		return fmt.Errorf("get product: %w", err)
	}
	if product.SellerID != sellerID {
		return fmt.Errorf("not owner")
	}
	return nil
}

func (s *ProductService) Delete(ctx context.Context, id uuid.UUID) (*generated.ProductResponse, error) {
	product, err := s.repo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrNotFound
		}
		return nil, fmt.Errorf("delete product: %w", err)
	}

	return toProductResponse(product), nil
}

func toProductResponse(p *model.Product) *generated.ProductResponse {
	return &generated.ProductResponse{
		Id:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		Stock:       p.Stock,
		Category:    p.Category,
		Status:      generated.ProductStatus(p.Status),
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}