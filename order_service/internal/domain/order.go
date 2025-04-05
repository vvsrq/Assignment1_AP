package domain

import "time"

type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusCompleted OrderStatus = "completed"
	StatusCancelled OrderStatus = "cancelled"
)

type Order struct {
	ID        int         `json:"id"`
	UserID    int         `json:"user_id"`
	Items     []OrderItem `json:"items"`
	Status    OrderStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ProductID int     `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type OrderRepository interface {
	CreateOrder(order *Order) (*Order, error)
	GetOrderByID(id int) (*Order, error)
	UpdateOrderStatus(id int, status OrderStatus) (*Order, error)
	ListOrdersByUserID(userID int, limit, offset int) ([]Order, error)
}

func IsValidStatus(status OrderStatus) bool {
	switch status {
	case StatusPending, StatusCompleted, StatusCancelled:
		return true
	default:
		return false
	}
}
