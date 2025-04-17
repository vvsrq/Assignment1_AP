package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"inventory_service/internal/domain"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type postgresCategoryRepository struct {
	db  *sql.DB
	log *logrus.Logger
}

func NewPostgresCategoryRepository(db *sql.DB, logger *logrus.Logger) domain.CategoryRepository {
	return &postgresCategoryRepository{
		db:  db,
		log: logger,
	}
}

func (r *postgresCategoryRepository) CreateCategory(category *domain.Category) (*domain.Category, error) {
	query := `INSERT INTO categories (name) VALUES ($1) returning id`
	err := r.db.QueryRow(query, category.Name).Scan(&category.ID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			r.log.Warnf("Attempted to create category with duplicate name: %s", category.Name)
			return nil, fmt.Errorf("category with name '%s' already exists", category.Name)
		}
		r.log.Errorf("Failed to create category '%s': %v", category.Name, err)
		return nil, fmt.Errorf("could not create category: %w", err)
	}
	r.log.Infof("Category created successfully with ID: %d, Name: %s", category.ID, category.Name)
	return category, nil
}

func (r *postgresCategoryRepository) GetCategoryByID(id int) (*domain.Category, error) {
	query := `SELECT id, name FROM categories WHERE id = $1`
	category := &domain.Category{}
	err := r.db.QueryRow(query, id).Scan(&category.ID, &category.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.Warnf("Category with ID %d not found", id)
			return nil, fmt.Errorf("category with id %d not found", id)
		}
		r.log.Errorf("Failed to get category by ID %d: %v", id, err)
		return nil, fmt.Errorf("could not get category by id: %w", err)
	}
	r.log.Infof("Category retrieved successfully with ID: %d", id)
	return category, nil
}

func (r *postgresCategoryRepository) UpdateCategory(category *domain.Category) (*domain.Category, error) {
	query := `UPDATE categories SET name = $1 WHERE id = $2 RETURNING id, name`
	err := r.db.QueryRow(query, category.Name, category.ID).Scan(&category.ID, &category.Name)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			r.log.Warnf("Attempted to update category ID %d with duplicate name: %s", category.ID, category.Name)
			return nil, fmt.Errorf("category with name '%s' already exists", category.Name)
		}
		if errors.Is(err, sql.ErrNoRows) {
			r.log.Warnf("Category with ID %d not found for update", category.ID)
			return nil, fmt.Errorf("category with id %d not found for update", category.ID)
		}
		r.log.Errorf("Failed to update category ID %d: %v", category.ID, err)
		return nil, fmt.Errorf("could not update category: %w", err)
	}
	r.log.Infof("Category updated successfully with ID: %d", category.ID)
	return category, nil
}

func (r *postgresCategoryRepository) DeleteCategory(id int) error {
	query := `DELETE FROM categories WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		r.log.Errorf("Failed to delete category ID %d: %v", id, err)
		return fmt.Errorf("could not delete category: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.log.Errorf("Failed to get rows affected after deleting category ID %d: %v", id, err)
		return fmt.Errorf("could not confirm category deletion: %w", err)
	}

	if rowsAffected == 0 {
		r.log.Warnf("Attempted to delete non-existent category ID %d", id)
		return fmt.Errorf("category with id %d not found for deletion", id)
	}

	r.log.Infof("Category deleted successfully with ID: %d", id)
	return nil
}

func (r *postgresCategoryRepository) ListCategories() ([]domain.Category, error) {
	query := `SELECT id, name FROM categories ORDER BY id ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		r.log.Errorf("Failed to list categories: %v", err)
		return nil, fmt.Errorf("could not list categories: %w", err)
	}
	defer rows.Close()

	categories := []domain.Category{}
	for rows.Next() {
		var category domain.Category
		if err := rows.Scan(&category.ID, &category.Name); err != nil {
			r.log.Errorf("Failed to scan category row: %v", err)
			continue
		}
		categories = append(categories, category)
	}

	if err = rows.Err(); err != nil {
		r.log.Errorf("Error during categories list iteration: %v", err)
		return nil, fmt.Errorf("error iterating categories: %w", err)
	}

	r.log.Infof("Retrieved %d categories", len(categories))
	return categories, nil
}
