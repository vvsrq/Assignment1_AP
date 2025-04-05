package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"order_service/internal/domain"
	_ "time"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type postgresOrderRepository struct {
	db  *sql.DB
	log *logrus.Logger
}

func NewPostgresOrderRepository(db *sql.DB, logger *logrus.Logger) domain.OrderRepository {
	return &postgresOrderRepository{
		db:  db,
		log: logger,
	}
}

func (r *postgresOrderRepository) CreateOrder(order *domain.Order) (*domain.Order, error) {
	tx, err := r.db.Begin()
	if err != nil {
		r.log.Errorf("Failed to begin transaction: %v", err)
		return nil, fmt.Errorf("could not start transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			r.log.Error("Recovered from panic, rolling back transaction")
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			r.log.Warnf("Rolling back transaction due to error: %v", err)
			if rbErr := tx.Rollback(); rbErr != nil {
				r.log.Errorf("Failed to rollback transaction: %v", rbErr)
			}
		} else {
			r.log.Info("Committing transaction")
			if cErr := tx.Commit(); cErr != nil {
				r.log.Errorf("Failed to commit transaction: %v", cErr)

				err = fmt.Errorf("failed to commit transaction: %w", cErr)

			}
		}
	}()

	orderQuery := `
        INSERT INTO orders (user_id, status)
        VALUES ($1, $2)
        RETURNING id, status, created_at, updated_at
    `
	err = tx.QueryRow(orderQuery, order.UserID, order.Status).Scan(
		&order.ID,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		r.log.Errorf("Failed to insert order for user %d: %v", order.UserID, err)

		return nil, fmt.Errorf("could not create order entry: %w", err)
	}
	r.log.Infof("Order entry created with ID: %d for user: %d", order.ID, order.UserID)

	itemQuery := `
        INSERT INTO order_items (order_id, product_id, quantity, price)
        VALUES ($1, $2, $3, $4)
        
    `
	stmt, err := tx.Prepare(itemQuery)
	if err != nil {
		r.log.Errorf("Failed to prepare order item statement: %v", err)
		return nil, fmt.Errorf("could not prepare item statement: %w", err)
	}
	defer stmt.Close()

	for i := range order.Items {
		item := &order.Items[i]
		_, err = stmt.Exec(order.ID, item.ProductID, item.Quantity, item.Price)
		if err != nil {
			r.log.Errorf("Failed to insert order item (product_id: %d, quantity: %d) for order %d: %v", item.ProductID, item.Quantity, order.ID, err)

			if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23514" {
				return nil, fmt.Errorf("invalid item data (product_id: %d): %s", item.ProductID, pqErr.Message)
			}
			return nil, fmt.Errorf("could not create order item (product_id: %d): %w", item.ProductID, err)
		}
		r.log.Infof("Order item inserted for order %d, product %d", order.ID, item.ProductID)
	}

	r.log.Infof("Order %d created successfully with %d items.", order.ID, len(order.Items))

	if err == nil && tx.Commit() != nil {
		return nil, err
	}

	return order, nil
}

func (r *postgresOrderRepository) GetOrderByID(id int) (*domain.Order, error) {
	order := &domain.Order{}
	orderQuery := `
        SELECT id, user_id, status, created_at, updated_at
        FROM orders
        WHERE id = $1
    `
	err := r.db.QueryRow(orderQuery, id).Scan(
		&order.ID,
		&order.UserID,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.Warnf("Order with ID %d not found", id)
			return nil, fmt.Errorf("order with id %d not found", id)
		}
		r.log.Errorf("Failed to get order by ID %d: %v", id, err)
		return nil, fmt.Errorf("could not retrieve order: %w", err)
	}

	items, err := r.getOrderItems(id)
	if err != nil {

		return nil, err
	}
	order.Items = items

	r.log.Infof("Order %d retrieved successfully with %d items.", order.ID, len(order.Items))
	return order, nil
}

func (r *postgresOrderRepository) getOrderItems(orderID int) ([]domain.OrderItem, error) {
	itemsQuery := `
        SELECT product_id, quantity, price
        FROM order_items
        WHERE order_id = $1
    `
	rows, err := r.db.Query(itemsQuery, orderID)
	if err != nil {
		r.log.Errorf("Failed to query order items for order ID %d: %v", orderID, err)
		return nil, fmt.Errorf("could not retrieve order items: %w", err)
	}
	defer rows.Close()

	var items []domain.OrderItem
	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(&item.ProductID, &item.Quantity, &item.Price); err != nil {
			r.log.Errorf("Failed to scan order item row for order ID %d: %v", orderID, err)

			return nil, fmt.Errorf("error scanning order item: %w", err)
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		r.log.Errorf("Error during order items iteration for order ID %d: %v", orderID, err)
		return nil, fmt.Errorf("error iterating order items: %w", err)
	}

	r.log.Debugf("Retrieved %d items for order ID %d", len(items), orderID)
	return items, nil
}

func (r *postgresOrderRepository) UpdateOrderStatus(id int, status domain.OrderStatus) (*domain.Order, error) {

	tx, err := r.db.Begin()
	if err != nil {
		r.log.Errorf("Failed to begin transaction for status update: %v", err)
		return nil, fmt.Errorf("could not start transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				r.log.Errorf("UpdateOrderStatus: Failed to rollback transaction: %v (original error: %v)", rbErr, err)
			}
		} else {
			if cErr := tx.Commit(); cErr != nil {
				err = fmt.Errorf("failed to commit status update transaction: %w", cErr)
				r.log.Errorf("UpdateOrderStatus: %v", err)
			}
		}
	}()

	query := `
        UPDATE orders
        SET status = $1, updated_at = NOW() 
        WHERE id = $2
        RETURNING id, user_id, status, created_at, updated_at
    `
	updatedOrder := &domain.Order{}

	err = tx.QueryRow(query, status, id).Scan(
		&updatedOrder.ID,
		&updatedOrder.UserID,
		&updatedOrder.Status,
		&updatedOrder.CreatedAt,
		&updatedOrder.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.Warnf("Order with ID %d not found for status update", id)

			return nil, fmt.Errorf("order with id %d not found for update", id)
		}

		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23514" { // check_violation или invalid enum
			r.log.Warnf("Invalid status value '%s' for order ID %d: %v", status, id, err)
			return nil, fmt.Errorf("invalid order status provided: %s", status)
		}
		r.log.Errorf("Failed to update status for order ID %d: %v", id, err)

		return nil, fmt.Errorf("could not update order status: %w", err)
	}

	items, err := r.getOrderItemsTx(tx, id)
	if err != nil {

		return nil, fmt.Errorf("order status updated, but failed to retrieve items: %w", err)
	}
	updatedOrder.Items = items

	r.log.Infof("Status and items retrieved successfully for order %d after update to '%s'.", updatedOrder.ID, updatedOrder.Status)

	return updatedOrder, nil
}

func (r *postgresOrderRepository) getOrderItemsTx(tx *sql.Tx, orderID int) ([]domain.OrderItem, error) {
	itemsQuery := `
        SELECT product_id, quantity, price
        FROM order_items
        WHERE order_id = $1
    `

	rows, err := tx.Query(itemsQuery, orderID)
	if err != nil {
		r.log.Errorf("Failed to query order items within tx for order ID %d: %v", orderID, err)
		return nil, fmt.Errorf("could not retrieve order items within tx: %w", err)
	}
	defer rows.Close()

	var items []domain.OrderItem
	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(&item.ProductID, &item.Quantity, &item.Price); err != nil {
			r.log.Errorf("Failed to scan order item row within tx for order ID %d: %v", orderID, err)
			return nil, fmt.Errorf("error scanning order item within tx: %w", err)
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		r.log.Errorf("Error during order items iteration within tx for order ID %d: %v", orderID, err)
		return nil, fmt.Errorf("error iterating order items within tx: %w", err)
	}

	r.log.Debugf("Retrieved %d items within tx for order ID %d", len(items), orderID)
	return items, nil
}

func (r *postgresOrderRepository) ListOrdersByUserID(userID int, limit, offset int) ([]domain.Order, error) {

	if limit <= 0 || limit > 100 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	ordersQuery := `
        SELECT id, user_id, status, created_at, updated_at
        FROM orders
        WHERE user_id = $1
        ORDER BY created_at DESC -- Сначала новые заказы
        LIMIT $2 OFFSET $3
    `
	rows, err := r.db.Query(ordersQuery, userID, limit, offset)
	if err != nil {
		r.log.Errorf("Failed to list orders for user ID %d: %v", userID, err)
		return nil, fmt.Errorf("could not retrieve orders: %w", err)
	}
	defer rows.Close()

	var orders []domain.Order
	orderIDs := []int{}

	for rows.Next() {
		var order domain.Order
		if err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
		); err != nil {
			r.log.Errorf("Failed to scan order row for user ID %d: %v", userID, err)
			return nil, fmt.Errorf("error scanning order data: %w", err)
		}
		orders = append(orders, order)
		orderIDs = append(orderIDs, order.ID)
	}
	if err = rows.Err(); err != nil {
		r.log.Errorf("Error during orders iteration for user ID %d: %v", userID, err)
		return nil, fmt.Errorf("error iterating orders: %w", err)
	}

	if len(orders) == 0 {
		r.log.Infof("No orders found for user ID %d with limit %d, offset %d", userID, limit, offset)
		return []domain.Order{}, nil
	}

	itemsQuery := `
        SELECT order_id, product_id, quantity, price
        FROM order_items
        WHERE order_id = ANY($1::int[]) -- Используем массив ID
		ORDER BY order_id -- Опционально, для группировки
    `

	itemRows, err := r.db.Query(itemsQuery, pq.Array(orderIDs))
	if err != nil {
		r.log.Errorf("Failed to query items for multiple orders (%v): %v", orderIDs, err)
		return nil, fmt.Errorf("could not retrieve order items for list: %w", err)
	}
	defer itemRows.Close()

	itemsMap := make(map[int][]domain.OrderItem)
	for itemRows.Next() {
		var item domain.OrderItem
		var orderID int
		if err := itemRows.Scan(&orderID, &item.ProductID, &item.Quantity, &item.Price); err != nil {
			r.log.Errorf("Failed to scan order item row during multi-order fetch: %v", err)
			return nil, fmt.Errorf("error scanning order item data for list: %w", err)
		}
		itemsMap[orderID] = append(itemsMap[orderID], item)
	}
	if err = itemRows.Err(); err != nil {
		r.log.Errorf("Error during multi-order items iteration: %v", err)
		return nil, fmt.Errorf("error iterating order items for list: %w", err)
	}

	for i := range orders {
		if items, ok := itemsMap[orders[i].ID]; ok {
			orders[i].Items = items
		} else {
			orders[i].Items = []domain.OrderItem{}
		}
	}

	r.log.Infof("Retrieved %d orders for user ID %d (limit %d, offset %d)", len(orders), userID, limit, offset)
	return orders, nil
}
