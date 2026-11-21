package payment

// The provider's wire format. These types never leave the package.

type chargeRequest struct {
	Reference string `json:"reference"`
	Amount    int64  `json:"amount"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
