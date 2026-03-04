package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shaiso/marketplace/generated"
	"github.com/shaiso/marketplace/internal/model"
	"github.com/shaiso/marketplace/internal/repo"
)

type OrderService struct {
	orderRepo     *repo.OrderRepo
	productRepo   *repo.ProductRepo
	promoRepo     *repo.PromoCodeRepo
	userOpRepo    *repo.UserOperationRepo
	rateLimitMins int
}

func NewOrderService(
	orderRepo *repo.OrderRepo,
	productRepo *repo.ProductRepo,
	promoRepo *repo.PromoCodeRepo,
	userOpRepo *repo.UserOperationRepo,
	rateLimitMins int,
) *OrderService {
	return &OrderService{
		orderRepo:     orderRepo,
		productRepo:   productRepo,
		promoRepo:     promoRepo,
		userOpRepo:    userOpRepo,
		rateLimitMins: rateLimitMins,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, userID uuid.UUID, req generated.OrderCreate) (*generated.OrderResponse, error) {

	if err := s.validateOrderCreate(req); err != nil {
		return nil, err
	}

	if err := s.checkRateLimit(ctx, userID, model.OperationCreateOrder); err != nil {
		return nil, err
	}

	_, err := s.orderRepo.GetActiveOrderByUserID(ctx, userID)
	if err == nil {
		return nil, &BusinessError{
			Code:    ErrCodeOrderHasActive,
			Message: "user already has an active order",
			Status:  http.StatusConflict,
		}
	}
	if !errors.Is(err, pgx.ErrNoRows) && !isNoRows(err) {
		return nil, fmt.Errorf("check active order: %w", err)
	}

	tx, err := s.orderRepo.Pool().Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	items, totalAmount, err := s.validateAndPrepareItems(ctx, tx, req.Items)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		if err := s.productRepo.UpdateStock(ctx, tx, item.ProductID, -item.Quantity); err != nil {
			return nil, fmt.Errorf("reserve stock: %w", err)
		}
	}

	var discountAmount float64
	var promoCodeID *uuid.UUID

	if req.PromoCode != nil && *req.PromoCode != "" {
		promo, discount, err := s.applyPromoCode(ctx, tx, *req.PromoCode, totalAmount)
		if err != nil {
			return nil, err
		}
		discountAmount = discount
		promoCodeID = &promo.ID
		totalAmount -= discount

		if err := s.promoRepo.IncrementUses(ctx, tx, promo.ID); err != nil {
			return nil, fmt.Errorf("increment promo uses: %w", err)
		}
	}

	order := model.Order{
		UserID:         userID,
		Status:         model.OrderStatusCreated,
		PromoCodeID:    promoCodeID,
		TotalAmount:    totalAmount,
		DiscountAmount: discountAmount,
	}

	if err := s.orderRepo.Create(ctx, tx, &order, items); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	userOp := model.UserOperation{
		UserID:        userID,
		OperationType: model.OperationCreateOrder,
	}
	if err := s.userOpRepo.Create(ctx, tx, &userOp); err != nil {
		return nil, fmt.Errorf("create user operation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return s.toOrderResponse(&order, items), nil
}

func (s *OrderService) GetOrder(ctx context.Context, userID, orderID uuid.UUID) (*generated.OrderResponse, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		if isNoRows(err) {
			return nil, &BusinessError{Code: ErrCodeOrderNotFound, Message: "order not found", Status: http.StatusNotFound}
		}
		return nil, fmt.Errorf("get order: %w", err)
	}

	if userID != uuid.Nil && order.UserID != userID {
		return nil, &BusinessError{Code: ErrCodeOrderOwnershipViolation, Message: "order belongs to another user", Status: http.StatusForbidden}
	}

	items, err := s.orderRepo.GetItemsByOrderID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("get order items: %w", err)
	}

	return s.toOrderResponse(order, items), nil
}

func (s *OrderService) UpdateOrder(ctx context.Context, userID, orderID uuid.UUID, req generated.OrderUpdate) (*generated.OrderResponse, error) {

	if err := s.validateOrderUpdate(req); err != nil {
		return nil, err
	}

	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		if isNoRows(err) {
			return nil, &BusinessError{Code: ErrCodeOrderNotFound, Message: "order not found", Status: http.StatusNotFound}
		}
		return nil, fmt.Errorf("get order: %w", err)
	}

	if userID != uuid.Nil && order.UserID != userID {
		return nil, &BusinessError{Code: ErrCodeOrderOwnershipViolation, Message: "order belongs to another user", Status: http.StatusForbidden}
	}

	if order.Status != model.OrderStatusCreated {
		return nil, &BusinessError{Code: ErrCodeInvalidStateTransition, Message: "order can only be updated in CREATED state", Status: http.StatusConflict}
	}

	if err := s.checkRateLimit(ctx, userID, model.OperationUpdateOrder); err != nil {
		return nil, err
	}

	tx, err := s.orderRepo.Pool().Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	oldItems, err := s.orderRepo.GetItemsByOrderIDWithDB(ctx, tx, orderID)
	if err != nil {
		return nil, fmt.Errorf("get old items: %w", err)
	}
	for _, item := range oldItems {
		if err := s.productRepo.UpdateStock(ctx, tx, item.ProductID, item.Quantity); err != nil {
			return nil, fmt.Errorf("restore stock: %w", err)
		}
	}

	newItems, totalAmount, err := s.validateAndPrepareItems(ctx, tx, req.Items)
	if err != nil {
		return nil, err
	}
	for _, item := range newItems {
		if err := s.productRepo.UpdateStock(ctx, tx, item.ProductID, -item.Quantity); err != nil {
			return nil, fmt.Errorf("reserve stock: %w", err)
		}
	}

	var discountAmount float64
	promoCodeID := order.PromoCodeID

	if promoCodeID != nil {
		promo, err := s.promoRepo.GetByID(ctx, tx, *promoCodeID)
		if err != nil {

			promoCodeID = nil
		} else {

			discount := s.calculateDiscount(promo, totalAmount)
			if totalAmount >= promo.MinOrderAmount && promo.Active {
				discountAmount = discount
				totalAmount -= discount
			} else {

				promoCodeID = nil
				if err := s.promoRepo.DecrementUses(ctx, tx, promo.ID); err != nil {
					return nil, fmt.Errorf("decrement promo uses: %w", err)
				}
			}
		}
	}

	order.TotalAmount = totalAmount
	order.DiscountAmount = discountAmount
	order.PromoCodeID = promoCodeID

	if err := s.orderRepo.UpdateOrder(ctx, tx, order, newItems); err != nil {
		return nil, fmt.Errorf("update order: %w", err)
	}

	userOp := model.UserOperation{
		UserID:        userID,
		OperationType: model.OperationUpdateOrder,
	}
	if err := s.userOpRepo.Create(ctx, tx, &userOp); err != nil {
		return nil, fmt.Errorf("create user operation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return s.toOrderResponse(order, newItems), nil
}

func (s *OrderService) CancelOrder(ctx context.Context, userID, orderID uuid.UUID) (*generated.OrderResponse, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		if isNoRows(err) {
			return nil, &BusinessError{Code: ErrCodeOrderNotFound, Message: "order not found", Status: http.StatusNotFound}
		}
		return nil, fmt.Errorf("get order: %w", err)
	}

	if userID != uuid.Nil && order.UserID != userID {
		return nil, &BusinessError{Code: ErrCodeOrderOwnershipViolation, Message: "order belongs to another user", Status: http.StatusForbidden}
	}

	if order.Status != model.OrderStatusCreated && order.Status != model.OrderStatusPaymentPending {
		return nil, &BusinessError{Code: ErrCodeInvalidStateTransition, Message: "order can only be canceled from CREATED or PAYMENT_PENDING state", Status: http.StatusConflict}
	}

	tx, err := s.orderRepo.Pool().Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	items, err := s.orderRepo.GetItemsByOrderIDWithDB(ctx, tx, orderID)
	if err != nil {
		return nil, fmt.Errorf("get order items: %w", err)
	}
	for _, item := range items {
		if err := s.productRepo.UpdateStock(ctx, tx, item.ProductID, item.Quantity); err != nil {
			return nil, fmt.Errorf("restore stock: %w", err)
		}
	}

	if order.PromoCodeID != nil {
		if err := s.promoRepo.DecrementUses(ctx, tx, *order.PromoCodeID); err != nil {
			return nil, fmt.Errorf("decrement promo uses: %w", err)
		}
	}

	if err := s.orderRepo.UpdateStatus(ctx, tx, orderID, model.OrderStatusCanceled); err != nil {
		return nil, fmt.Errorf("cancel order: %w", err)
	}
	order.Status = model.OrderStatusCanceled

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return s.toOrderResponse(order, items), nil
}

func (s *OrderService) checkRateLimit(ctx context.Context, userID uuid.UUID, opType model.OperationType) error {
	lastOp, err := s.userOpRepo.GetLastByType(ctx, userID, opType)
	if err != nil {
		if isNoRows(err) {
			return nil // Нет предыдущих операций — лимит не превышен
		}
		return fmt.Errorf("check rate limit: %w", err)
	}

	if time.Since(lastOp.CreatedAt) < time.Duration(s.rateLimitMins)*time.Minute {
		return &BusinessError{
			Code:    ErrCodeOrderLimitExceeded,
			Message: fmt.Sprintf("please wait %d minutes between operations", s.rateLimitMins),
			Status:  http.StatusTooManyRequests,
		}
	}
	return nil
}

func (s *OrderService) validateAndPrepareItems(ctx context.Context, db repo.DBTX, reqItems []generated.OrderItemRequest) ([]model.OrderItem, float64, error) {
	var items []model.OrderItem
	var totalAmount float64
	var insufficientStock []map[string]interface{}

	for _, reqItem := range reqItems {
		product, err := s.productRepo.GetByIDWithDB(ctx, db, reqItem.ProductId)
		if err != nil {
			if isNoRows(err) {
				return nil, 0, &BusinessError{
					Code:    ErrCodeProductNotFound,
					Message: fmt.Sprintf("product %s not found", reqItem.ProductId),
					Status:  http.StatusConflict,
				}
			}
			return nil, 0, fmt.Errorf("get product: %w", err)
		}

		if product.Status != model.ProductStatusActive {
			return nil, 0, &BusinessError{
				Code:    ErrCodeProductInactive,
				Message: fmt.Sprintf("product %s is not active", reqItem.ProductId),
				Status:  http.StatusConflict,
			}
		}

		if product.Stock < reqItem.Quantity {
			insufficientStock = append(insufficientStock, map[string]interface{}{
				"product_id": reqItem.ProductId.String(),
				"requested":  reqItem.Quantity,
				"available":  product.Stock,
			})
			continue
		}

		priceAtOrder := product.Price
		totalAmount += priceAtOrder * float64(reqItem.Quantity)

		items = append(items, model.OrderItem{
			ProductID:    reqItem.ProductId,
			Quantity:     reqItem.Quantity,
			PriceAtOrder: priceAtOrder,
		})
	}

	if len(insufficientStock) > 0 {
		return nil, 0, &BusinessError{
			Code:    ErrCodeInsufficientStock,
			Message: "insufficient stock for some products",
			Status:  http.StatusConflict,
			Details: map[string]interface{}{"products": insufficientStock},
		}
	}

	return items, totalAmount, nil
}

func (s *OrderService) applyPromoCode(ctx context.Context, db repo.DBTX, code string, totalAmount float64) (*model.PromoCode, float64, error) {
	promo, err := s.promoRepo.GetByCode(ctx, db, code)
	if err != nil {
		if isNoRows(err) {
			return nil, 0, &BusinessError{Code: ErrCodePromoCodeInvalid, Message: "promo code not found", Status: http.StatusUnprocessableEntity}
		}
		return nil, 0, fmt.Errorf("get promo code: %w", err)
	}

	now := time.Now()
	if !promo.Active || promo.CurrentUses >= promo.MaxUses || now.Before(promo.ValidFrom) || now.After(promo.ValidUntil) {
		return nil, 0, &BusinessError{Code: ErrCodePromoCodeInvalid, Message: "promo code is not valid", Status: http.StatusUnprocessableEntity}
	}

	if totalAmount < promo.MinOrderAmount {
		return nil, 0, &BusinessError{
			Code:    ErrCodePromoCodeMinAmount,
			Message: fmt.Sprintf("order total %.2f is less than minimum %.2f", totalAmount, promo.MinOrderAmount),
			Status:  http.StatusUnprocessableEntity,
		}
	}

	discount := s.calculateDiscount(promo, totalAmount)
	return promo, discount, nil
}

func (s *OrderService) calculateDiscount(promo *model.PromoCode, totalAmount float64) float64 {
	switch promo.DiscountType {
	case model.DiscountPercentage:
		discount := totalAmount * promo.DiscountValue / 100
		maxDiscount := totalAmount * 0.7
		return math.Min(discount, maxDiscount)
	case model.DiscountFixedAmount:
		return math.Min(promo.DiscountValue, totalAmount)
	default:
		return 0
	}
}

func (s *OrderService) validateOrderCreate(req generated.OrderCreate) error {
	fields := make(map[string]string)

	if len(req.Items) == 0 {
		fields["items"] = "must have at least 1 item"
	}
	if len(req.Items) > 50 {
		fields["items"] = "must have at most 50 items"
	}

	for i, item := range req.Items {
		if item.Quantity < 1 || item.Quantity > 999 {
			fields[fmt.Sprintf("items[%d].quantity", i)] = "must be between 1 and 999"
		}
	}

	if req.PromoCode != nil && *req.PromoCode != "" {
		code := *req.PromoCode
		if len(code) < 4 || len(code) > 20 {
			fields["promo_code"] = "must be between 4 and 20 characters"
		}
	}

	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

func (s *OrderService) validateOrderUpdate(req generated.OrderUpdate) error {
	fields := make(map[string]string)

	if len(req.Items) == 0 {
		fields["items"] = "must have at least 1 item"
	}
	if len(req.Items) > 50 {
		fields["items"] = "must have at most 50 items"
	}

	for i, item := range req.Items {
		if item.Quantity < 1 || item.Quantity > 999 {
			fields[fmt.Sprintf("items[%d].quantity", i)] = "must be between 1 and 999"
		}
	}

	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

func (s *OrderService) toOrderResponse(order *model.Order, items []model.OrderItem) *generated.OrderResponse {
	respItems := make([]generated.OrderItemResponse, len(items))
	for i, item := range items {
		respItems[i] = generated.OrderItemResponse{
			Id:           item.ID,
			OrderId:      item.OrderID,
			ProductId:    item.ProductID,
			Quantity:     item.Quantity,
			PriceAtOrder: item.PriceAtOrder,
		}
	}

	resp := &generated.OrderResponse{
		Id:             order.ID,
		UserId:         order.UserID,
		Status:         generated.OrderStatus(order.Status),
		TotalAmount:    order.TotalAmount,
		DiscountAmount: order.DiscountAmount,
		Items:          respItems,
		CreatedAt:      order.CreatedAt,
		UpdatedAt:      order.UpdatedAt,
	}

	if order.PromoCodeID != nil {
		resp.PromoCodeId = order.PromoCodeID
	}

	return resp
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
