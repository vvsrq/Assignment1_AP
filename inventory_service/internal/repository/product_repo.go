package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"inventory_service/internal/domain"
	"strings"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type postgresProductRepository struct {
	db  *sql.DB
	log *logrus.Logger
}

func NewPostgresProductRepository(db *sql.DB, logger *logrus.Logger) domain.ProductRepository {
	return &postgresProductRepository{
		db:  db,
		log: logger,
	}
}

func (r *postgresProductRepository) CreateProduct(product *domain.Product) (*domain.Product, error) {
	query := `
        INSERT INTO products (name, price, stock, category_id)
        VALUES ($1, $2, $3, $4)
        RETURNING id`
	var categoryID sql.NullInt64
	if product.CategoryID != 0 {
		categoryID = sql.NullInt64{Int64: int64(product.CategoryID), Valid: true}
	} else {
		categoryID = sql.NullInt64{Valid: false}
	}

	err := r.db.QueryRow(query, product.Name, product.Price, product.Stock, categoryID).Scan(&product.ID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			r.log.Warnf("Attempted to create product with non-existent category ID: %d", product.CategoryID)
			return nil, fmt.Errorf("category with id %d does not exist", product.CategoryID)
		}
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23514" {
			r.log.Warnf("Check constraint violation for product '%s': %s", product.Name, pqErr.Message)
			return nil, fmt.Errorf("product data constraint violation: %s", pqErr.Message)
		}
		r.log.Errorf("Failed to create product '%s': %v", product.Name, err)
		return nil, fmt.Errorf("could not create product: %w", err)
	}
	r.log.Infof("Product created successfully with ID: %d, Name: %s", product.ID, product.Name)
	return product, nil
}

func (r *postgresProductRepository) GetProductByID(id int) (*domain.Product, error) {
	query := `
        SELECT id, name, price, stock, category_id
        FROM products
        WHERE id = $1`
	product := &domain.Product{}
	var categoryID sql.NullInt64

	err := r.db.QueryRow(query, id).Scan(
		&product.ID,
		&product.Name,
		&product.Price,
		&product.Stock,
		&categoryID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.Warnf("Product with ID %d not found", id)
			return nil, fmt.Errorf("product with id %d not found", id)
		}
		r.log.Errorf("Failed to get product by ID %d: %v", id, err)
		return nil, fmt.Errorf("could not get product by id: %w", err)
	}

	if categoryID.Valid {
		product.CategoryID = int(categoryID.Int64)
	} else {
		product.CategoryID = 0
	}

	r.log.Infof("Product retrieved successfully with ID: %d", id)
	return product, nil
}

func (r *postgresProductRepository) UpdateProduct(id int, updates map[string]interface{}) (*domain.Product, error) {
	if len(updates) == 0 {
		r.log.Info("Repository: No fields provided for product update ID %d. Returning current product.", id)
		return r.GetProductByID(id)
	}

	queryBase := "UPDATE products SET "
	args := []interface{}{}
	setClauses := []string{}
	argCounter := 1

	for key, value := range updates {
		column := ""
		argValue := value

		switch key {
		case "name":
			column = "name"
		case "price":
			column = "price"
		case "stock":
			column = "stock"
		case "category_id":
			column = "category_id"

			catID, ok := value.(int)
			if !ok {
				r.log.Errorf("Repository: Invalid type received for category_id for product ID %d: %T", id, value)
				return nil, fmt.Errorf("internal error: invalid type for category_id in repository")
			}
			if catID == 0 {
				argValue = nil
			} else {
				argValue = catID
			}
		default:

			r.log.Warnf("Repository: Skipping unknown field '%s' provided for product update ID %d", key, id)
			continue
		}

		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", column, argCounter))
		args = append(args, argValue)
		argCounter++
	}

	if len(setClauses) == 0 {
		r.log.Warn("Repository: No valid known fields provided for product update ID %d. Returning current product.", id)
		return r.GetProductByID(id)
	}

	query := queryBase + strings.Join(setClauses, ", ") + fmt.Sprintf(" WHERE id = $%d", argCounter)
	args = append(args, id) // Добавляем ID в конец аргументов

	r.log.Debugf("Repository: Executing partial update query for ID %d: %s with args: %v", id, query, args)

	result, err := r.db.Exec(query, args...)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			catID := 0
			if catIDVal, exists := updates["category_id"]; exists {
				catID, _ = catIDVal.(int)
			}
			r.log.Warnf("Repository: Attempted to update product ID %d with non-existent category ID: %d", id, catID)
			return nil, fmt.Errorf("category with id %d does not exist", catID)
		}

		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23514" {
			r.log.Warnf("Repository: Check constraint violation for product update ID %d: %s", id, pqErr.Message)
			return nil, fmt.Errorf("product data constraint violation: %s", pqErr.Message)
		}
		r.log.Errorf("Repository: Failed to execute partial update for product ID %d: %v", id, err)
		return nil, fmt.Errorf("could not partially update product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.log.Errorf("Repository: Failed to get rows affected after partial update for ID %d: %v", id, err)

	}

	if rowsAffected == 0 {
		r.log.Warnf("Repository: Product with ID %d not found for update (0 rows affected)", id)
		return nil, fmt.Errorf("product with id %d not found for update", id)
	}

	r.log.Infof("Repository: Partial update successful for product ID %d (%d rows affected). Fetching updated product.", id, rowsAffected)
	return r.GetProductByID(id)
}

func (r *postgresProductRepository) DeleteProduct(id int) error {
	query := `DELETE FROM products WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		r.log.Errorf("Failed to delete product ID %d: %v", id, err)
		return fmt.Errorf("could not delete product: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.log.Errorf("Failed to get rows affected after deleting product ID %d: %v", id, err)
		return fmt.Errorf("could not confirm product deletion: %w", err)
	}
	if rowsAffected == 0 {
		r.log.Warnf("Attempted to delete non-existent product ID %d", id)
		return fmt.Errorf("product with id %d not found for deletion", id)
	}
	r.log.Infof("Product deleted successfully with ID: %d", id)
	return nil
}

func (r *postgresProductRepository) ListProducts(limit, offset int) ([]domain.Product, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	query := `
        SELECT id, name, price, stock, category_id
        FROM products
        ORDER BY id ASC
        LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		r.log.Errorf("Failed to list products with limit %d, offset %d: %v", limit, offset, err)
		return nil, fmt.Errorf("could not list products: %w", err)
	}
	defer rows.Close()

	products := []domain.Product{}
	for rows.Next() {
		var product domain.Product
		var categoryID sql.NullInt64
		if err := rows.Scan(&product.ID, &product.Name, &product.Price, &product.Stock, &categoryID); err != nil {
			r.log.Errorf("Failed to scan product row: %v", err)
			return nil, fmt.Errorf("error scanning product data: %w", err)
		}
		product.CategoryID = 0
		if categoryID.Valid {
			product.CategoryID = int(categoryID.Int64)
		}
		products = append(products, product)
	}
	if err = rows.Err(); err != nil {
		r.log.Errorf("Error during products list iteration: %v", err)
		return nil, fmt.Errorf("error iterating products: %w", err)
	}
	r.log.Infof("Retrieved %d products (limit: %d, offset: %d)", len(products), limit, offset)
	return products, nil
}

func (r *postgresProductRepository) ListProductsByCategory(categoryID, limit, offset int) ([]domain.Product, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	query := `
        SELECT id, name, price, stock, category_id
        FROM products
        WHERE category_id = $1
        ORDER BY id ASC
        LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(query, categoryID, limit, offset)
	if err != nil {
		r.log.Errorf("Failed to list products for category %d (limit %d, offset %d): %v", categoryID, limit, offset, err)
		return nil, fmt.Errorf("could not list products by category: %w", err)
	}
	defer rows.Close()

	products := []domain.Product{}
	for rows.Next() {
		var product domain.Product
		var catID sql.NullInt64
		if err := rows.Scan(&product.ID, &product.Name, &product.Price, &product.Stock, &catID); err != nil {
			r.log.Errorf("Failed to scan product row for category %d: %v", categoryID, err)
			return nil, fmt.Errorf("error scanning product data for category: %w", err)
		}
		product.CategoryID = 0
		if catID.Valid {
			product.CategoryID = int(catID.Int64)
		}
		products = append(products, product)
	}
	if err = rows.Err(); err != nil {
		r.log.Errorf("Error during products by category list iteration: %v", err)
		return nil, fmt.Errorf("error iterating products by category: %w", err)
	}
	r.log.Infof("Retrieved %d products for category %d (limit: %d, offset: %d)", len(products), categoryID, limit, offset)
	return products, nil
}
