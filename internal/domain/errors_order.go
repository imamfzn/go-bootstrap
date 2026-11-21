package domain

var (
	ErrOrderNotFound         = newError("ORDER_GET_NOT-FOUND", "Order not found")
	ErrOrderAlreadyExists    = newError("ORDER_CREATE_ALREADY-EXISTS", "Order already exists")
	ErrOrderProductRequired  = newError("ORDER_CREATE_PRODUCT-REQUIRED", "A product is required")
	ErrOrderQuantityInvalid  = newError("ORDER_CREATE_QUANTITY-INVALID", "Quantity must be between 1 and 100")
	ErrOrderAmountInvalid    = newError("ORDER_CREATE_AMOUNT-INVALID", "Amount must be greater than zero")
	ErrOrderNoteTooLong      = newError("ORDER_CREATE_NOTE-TOO-LONG", "Note must be 500 characters or fewer")
	ErrOrderCancelNotAllowed = newError("ORDER_CANCEL_NOT-ALLOWED", "This order can no longer be cancelled")
	ErrOrderPayNotAllowed    = newError("ORDER_PAY_NOT-ALLOWED", "This order can no longer be paid")
	ErrOrderPaymentDeclined  = newError("ORDER_PAY_PAYMENT-DECLINED", "The payment was declined")
)
