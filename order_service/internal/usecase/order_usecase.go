package usecase

import (
	"context"
	"errors"
	"fmt"
	"order_service/internal/clients"
	"order_service/internal/domain"

	"github.com/sirupsen/logrus"
)

var _ domain.OrderUseCase = (*orderUseCase)(nil)

type orderUseCase struct {
	orderRepo       domain.OrderRepository
	inventoryClient clients.InventoryClient
	log             *logrus.Logger
}

func NewOrderUseCase(repo domain.OrderRepository, invClient clients.InventoryClient, logger *logrus.Logger) domain.OrderUseCase {
	return &orderUseCase{
		orderRepo:       repo,
		inventoryClient: invClient,
		log:             logger,
	}
}

type productCheckInfo struct {
	Product       *clients.Product
	OrderQuantity int
}

func (uc *orderUseCase) CreateOrder(ctx context.Context, order *domain.Order) (*domain.Order, error) {

	if order.UserID <= 0 {
		return nil, errors.New("invalid user ID")
	}
	if len(order.Items) == 0 {
		return nil, errors.New("order must contain at least one item")
	}
	for i, item := range order.Items {
		if item.ProductID <= 0 {
			return nil, fmt.Errorf("item %d: invalid product ID", i)
		}
		if item.Quantity <= 0 {
			return nil, fmt.Errorf("item %d (product %d): quantity must be positive", i, item.ProductID)
		}

		if item.Price < 0 {
			return nil, fmt.Errorf("item %d (product %d): price cannot be negative", i, item.ProductID)
		}
	}
	if order.Status == "" {
		order.Status = domain.StatusPending
	}
	if order.Status != domain.StatusPending {
		return nil, fmt.Errorf("order can only be created with '%s' status", domain.StatusPending)
	}
	uc.log.Infof("Use Case: Validated basic order data for user %d. Status set to %s.", order.UserID, order.Status)

	uc.log.Infof("Use Case: Starting inventory check and reservation for order (user %d)", order.UserID)

	productsInfo := make(map[int]productCheckInfo)

	for i, item := range order.Items {
		uc.log.Infof("Use Case: Checking inventory for Product ID: %d (Quantity: %d)", item.ProductID, item.Quantity)

		product, err := uc.inventoryClient.GetProduct(ctx, item.ProductID)
		if err != nil {
			uc.log.Warnf("Use Case: Inventory check failed for Product ID %d: %v", item.ProductID, err)

			return nil, fmt.Errorf("inventory check failed for product %d: %w", item.ProductID, err)
		}

		order.Items[i].Price = product.Price
		uc.log.Infof("Use Case: Updated item price for Product ID %d to %.2f from inventory", item.ProductID, product.Price)

		currentRequested := item.Quantity
		if existing, ok := productsInfo[item.ProductID]; ok {
			currentRequested += existing.OrderQuantity
		}

		if product.Stock < currentRequested {
			uc.log.Warnf("Use Case: Insufficient stock for Product ID %d (Requested total: %d, Available: %d)", item.ProductID, currentRequested, product.Stock)
			return nil, fmt.Errorf("insufficient stock for product %d (requested total: %d, available: %d)", item.ProductID, currentRequested, product.Stock)
		}

		if existing, ok := productsInfo[item.ProductID]; ok {
			existing.OrderQuantity += item.Quantity
			productsInfo[item.ProductID] = existing
		} else {
			productsInfo[item.ProductID] = productCheckInfo{
				Product:       product,
				OrderQuantity: item.Quantity,
			}
		}
		uc.log.Infof("Use Case: Inventory check OK for Product ID %d (Stock: %d >= Requested: %d)", item.ProductID, product.Stock, productsInfo[item.ProductID].OrderQuantity)
	}

	successfullyDecreased := make(map[int]int)

	for productID, info := range productsInfo {
		newStock := info.Product.Stock - info.OrderQuantity
		uc.log.Infof("Use Case: Attempting to decrease stock via gRPC for Product ID %d from %d to %d", productID, info.Product.Stock, newStock)

		err := uc.inventoryClient.UpdateStock(ctx, productID, newStock)
		if err != nil {
			uc.log.Errorf("Use Case: Failed to decrease stock for Product ID %d via gRPC: %v. Rolling back...", productID, err)

			uc.log.Warnf("Use Case: Rolling back inventory changes due to error.")
			for idToRollback, quantityDecreased := range successfullyDecreased {
				currentInfoToRollback := productsInfo[idToRollback]
				quantityDecreased++
				rollbackStock := currentInfoToRollback.Product.Stock
				uc.log.Warnf("Use Case: Rolling back Product ID %d to stock %d via gRPC", idToRollback, rollbackStock)

				if rollbackErr := uc.inventoryClient.UpdateStock(ctx, idToRollback, rollbackStock); rollbackErr != nil {
					uc.log.Errorf("Use Case: CRITICAL! Failed to rollback stock via gRPC for Product ID %d: %v. Manual intervention required!", idToRollback, rollbackErr)

				}
			}
			return nil, fmt.Errorf("failed to reserve stock for product %d: %w", productID, err)
		}
		successfullyDecreased[productID] = info.OrderQuantity
		uc.log.Infof("Use Case: Successfully decreased stock via gRPC for Product ID %d", productID)
	}

	uc.log.Info("Use Case: Inventory reservation successful.")

	uc.log.Infof("Use Case: Attempting to save order for user %d to repository.", order.UserID)
	createdOrder, err := uc.orderRepo.CreateOrder(order)
	if err != nil {
		uc.log.Errorf("Use Case: Repository failed to create order for user %d AFTER inventory update: %v. Attempting rollback...", order.UserID, err)

		uc.log.Warnf("Use Case: Rolling back inventory changes due to DB error.")
		for idToRollback, quantityDecreased := range successfullyDecreased {
			quantityDecreased++
			currentInfoToRollback := productsInfo[idToRollback]
			rollbackStock := currentInfoToRollback.Product.Stock
			uc.log.Warnf("Use Case: Rolling back Product ID %d to stock %d via gRPC due to DB error", idToRollback, rollbackStock)

			if rollbackErr := uc.inventoryClient.UpdateStock(ctx, idToRollback, rollbackStock); rollbackErr != nil {
				uc.log.Errorf("Use Case: CRITICAL! Failed to rollback stock via gRPC for Product ID %d after DB error: %v. Manual intervention required!", idToRollback, rollbackErr)
			}
		}
		return nil, fmt.Errorf("failed to save order after reserving stock: %w", err)
	}

	uc.log.Infof("Use Case: Order created successfully with ID %d for user %d", createdOrder.ID, createdOrder.UserID)
	return createdOrder, nil
}

func (uc *orderUseCase) GetOrderByID(id int) (*domain.Order, error) {
	if id <= 0 {
		return nil, errors.New("invalid order ID")
	}
	uc.log.Infof("Use Case: Attempting to get order with ID %d", id)
	order, err := uc.orderRepo.GetOrderByID(id)
	if err != nil {
		uc.log.Warnf("Use Case: Repository failed to get order ID %d: %v", id, err)
		return nil, err
	}
	uc.log.Infof("Use Case: Order retrieved successfully for ID %d", id)
	return order, nil
}

func (uc *orderUseCase) UpdateOrderStatus(ctx context.Context, id int, status domain.OrderStatus) (*domain.Order, error) {

	if id <= 0 {
		return nil, errors.New("invalid order ID for status update")
	}
	if !domain.IsValidStatus(status) {
		return nil, fmt.Errorf("invalid target order status: %s", status)
	}

	uc.log.Infof("Use Case: Attempting to update status for order ID %d to '%s'", id, status)

	currentOrder, err := uc.orderRepo.GetOrderByID(id)
	if err != nil {
		uc.log.Warnf("Use Case: Could not get current order %d for status update check: %v", id, err)
		return nil, err
	}
	uc.log.Infof("Use Case: Current status for order %d is '%s'", id, currentOrder.Status)

	if currentOrder.Status == domain.StatusCompleted && status == domain.StatusCancelled {
		uc.log.Warnf("Use Case: Attempt to cancel an already completed order %d", id)
		return nil, errors.New("cannot cancel a completed order")
	}
	if currentOrder.Status == domain.StatusCancelled && status != domain.StatusCancelled {
		uc.log.Warnf("Use Case: Attempt to change status of an already cancelled order %d", id)
		return nil, errors.New("cannot change status of a cancelled order")
	}

	isCancelling := status == domain.StatusCancelled && currentOrder.Status != domain.StatusCancelled
	if isCancelling {
		uc.log.Infof("Use Case: Order %d is being cancelled. Returning items to inventory via gRPC.", id)
		for _, item := range currentOrder.Items {

			product, err := uc.inventoryClient.GetProduct(ctx, item.ProductID)
			if err != nil {

				uc.log.Errorf("Use Case: CRITICAL! Failed to get product %d info via gRPC to return stock for cancelled order %d: %v. Manual stock adjustment needed!", item.ProductID, id, err)
				continue
			}

			newStock := product.Stock + item.Quantity
			uc.log.Warnf("Use Case: Returning stock via gRPC for Product ID %d (Order: %d). Current: %d, Quantity: %d, New: %d", item.ProductID, id, product.Stock, item.Quantity, newStock)

			err = uc.inventoryClient.UpdateStock(ctx, item.ProductID, newStock)
			if err != nil {

				uc.log.Errorf("Use Case: CRITICAL! Failed to return stock via gRPC for Product ID %d (quantity %d) for cancelled order %d: %v. Manual stock adjustment needed!", item.ProductID, item.Quantity, id, err)
			} else {
				uc.log.Infof("Use Case: Successfully returned stock via gRPC for Product ID %d", item.ProductID)
			}
		}
	}

	uc.log.Infof("Use Case: Attempting to update order status in repository for ID %d to '%s'", id, status)
	updatedOrder, err := uc.orderRepo.UpdateOrderStatus(id, status)
	if err != nil {
		uc.log.Errorf("Use Case: Repository failed to update status for order ID %d: %v", id, err)

		if isCancelling {
			uc.log.Errorf("Use Case: WARNING! Failed to update order status to CANCELLED in DB after attempting inventory stock return for order %d. Potential inconsistency!", id)
		}
		return nil, err
	}

	uc.log.Infof("Use Case: Order status updated successfully for ID %d to %s", updatedOrder.ID, updatedOrder.Status)
	return updatedOrder, nil
}

func (uc *orderUseCase) ListOrdersByUserID(userID int, limit, offset int) ([]domain.Order, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user ID")
	}

	uc.log.Infof("Use Case: Attempting to list orders for user %d (limit: %d, offset: %d)", userID, limit, offset)
	orders, err := uc.orderRepo.ListOrdersByUserID(userID, limit, offset)
	if err != nil {
		uc.log.Errorf("Use Case: Repository failed to list orders for user %d: %v", userID, err)
		return nil, fmt.Errorf("could not retrieve orders for user %d: %w", userID, err)
	}

	uc.log.Infof("Use Case: Retrieved %d orders for user %d", len(orders), userID)
	return orders, nil
}
