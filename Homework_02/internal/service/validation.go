package service

import (
	"fmt"

	"github.com/shaiso/marketplace/generated"
)

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return "validation error"
}

func validateProductCreate(req generated.ProductCreate) error {
	fields := make(map[string]string)

	if len(req.Name) < 1 || len(req.Name) > 255 {
		fields["name"] = fmt.Sprintf("must be between 1 and 255 characters, got %d", len(req.Name))
	}

	if req.Description != nil && len(*req.Description) > 4000 {
		fields["description"] = fmt.Sprintf("must be at most 4000 characters, got %d", len(*req.Description))
	}

	if req.Price < 0.01 {
		fields["price"] = "must be greater than 0"
	}

	if req.Stock < 0 {
		fields["stock"] = "must be >= 0"
	}

	if len(req.Category) < 1 || len(req.Category) > 100 {
		fields["category"] = fmt.Sprintf("must be between 1 and 100 characters, got %d", len(req.Category))
	}

	if !req.Status.Valid() {
		fields["status"] = "must be one of: ACTIVE, INACTIVE, ARCHIVED"
	}

	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

func validateProductUpdate(req generated.ProductUpdate) error {
	fields := make(map[string]string)

	if len(req.Name) < 1 || len(req.Name) > 255 {
		fields["name"] = fmt.Sprintf("must be between 1 and 255 characters, got %d", len(req.Name))
	}

	if req.Description != nil && len(*req.Description) > 4000 {
		fields["description"] = fmt.Sprintf("must be at most 4000 characters, got %d", len(*req.Description))
	}

	if req.Price < 0.01 {
		fields["price"] = "must be greater than 0"
	}

	if req.Stock < 0 {
		fields["stock"] = "must be >= 0"
	}

	if len(req.Category) < 1 || len(req.Category) > 100 {
		fields["category"] = fmt.Sprintf("must be between 1 and 100 characters, got %d", len(req.Category))
	}

	if !req.Status.Valid() {
		fields["status"] = "must be one of: ACTIVE, INACTIVE, ARCHIVED"
	}

	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}
