package handler

// Handler объединяет все хэндлеры и реализует generated.StrictServerInterface.
type Handler struct {
	*ProductHandler
	*OrderHandler
	*PromoCodeHandler
	*AuthHandler
}

func NewHandler(product *ProductHandler, order *OrderHandler, promo *PromoCodeHandler, auth *AuthHandler) *Handler {
	return &Handler{
		ProductHandler:   product,
		OrderHandler:     order,
		PromoCodeHandler: promo,
		AuthHandler:      auth,
	}
}
