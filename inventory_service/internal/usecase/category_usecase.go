package usecase

import (
	"errors"
	"fmt"

	"inventory_service/internal/domain"

	"github.com/sirupsen/logrus"
)

type CategoryUseCase interface {
	CreateCategory(category *domain.Category) (*domain.Category, error)
	GetCategoryByID(id int) (*domain.Category, error)
	UpdateCategory(category *domain.Category) (*domain.Category, error)
	DeleteCategory(id int) error
	ListCategories() ([]domain.Category, error)
}

type categoryUseCase struct {
	categoryRepo domain.CategoryRepository
	log          *logrus.Logger
}

func NewCategoryUseCase(repo domain.CategoryRepository, logger *logrus.Logger) CategoryUseCase {
	return &categoryUseCase{
		categoryRepo: repo,
		log:          logger,
	}
}

func (uc *categoryUseCase) CreateCategory(category *domain.Category) (*domain.Category, error) {
	if category.Name == "" {
		uc.log.Warn("Use Case: Attempted to create category with empty name")
		return nil, errors.New("category name cannot be empty")
	}

	uc.log.Infof("Use Case: Attempting to create category with name '%s'", category.Name)
	createdCategory, err := uc.categoryRepo.CreateCategory(category)
	if err != nil {
		uc.log.Errorf("Use Case: Repository failed to create category '%s': %v", category.Name, err)
		return nil, err
	}

	uc.log.Infof("Use Case: Category '%s' created successfully with ID %d", createdCategory.Name, createdCategory.ID)
	return createdCategory, nil
}

func (uc *categoryUseCase) GetCategoryByID(id int) (*domain.Category, error) {
	if id <= 0 {
		uc.log.Warnf("Use Case: Attempted to get category with invalid ID: %d", id)
		return nil, errors.New("invalid category ID")
	}

	uc.log.Infof("Use Case: Attempting to get category with ID %d", id)
	category, err := uc.categoryRepo.GetCategoryByID(id)
	if err != nil {
		uc.log.Warnf("Use Case: Repository failed to get category ID %d: %v", id, err)
		return nil, err
	}

	uc.log.Infof("Use Case: Category retrieved successfully for ID %d", id)
	return category, nil
}

func (uc *categoryUseCase) UpdateCategory(category *domain.Category) (*domain.Category, error) {
	if category.ID <= 0 {
		uc.log.Warnf("Use Case: Attempted update with invalid ID: %d", category.ID)
		return nil, errors.New("invalid category ID for update")
	}
	if category.Name == "" {
		uc.log.Warnf("Use Case: Attempted update for ID %d with empty name", category.ID)
		return nil, errors.New("category name cannot be empty for update")
	}

	uc.log.Infof("Use Case: Attempting to update category ID %d", category.ID)
	updatedCategory, err := uc.categoryRepo.UpdateCategory(category)
	if err != nil {
		uc.log.Errorf("Use Case: Repository failed to update category ID %d: %v", category.ID, err)
		return nil, err
	}

	uc.log.Infof("Use Case: Category updated successfully for ID %d", updatedCategory.ID)
	return updatedCategory, nil
}

func (uc *categoryUseCase) DeleteCategory(id int) error {
	if id <= 0 {
		uc.log.Warnf("Use Case: Attempted delete with invalid ID: %d", id)
		return errors.New("invalid category ID for delete")
	}

	uc.log.Infof("Use Case: Attempting to delete category ID %d", id)
	err := uc.categoryRepo.DeleteCategory(id)
	if err != nil {
		uc.log.Warnf("Use Case: Repository failed to delete category ID %d: %v", id, err)
		return err
	}

	uc.log.Infof("Use Case: Category deleted successfully for ID %d", id)
	return nil
}

func (uc *categoryUseCase) ListCategories() ([]domain.Category, error) {
	uc.log.Info("Use Case: Attempting to list all categories")

	categories, err := uc.categoryRepo.ListCategories()
	if err != nil {
		uc.log.Errorf("Use Case: Repository failed to list categories: %v", err)

		return nil, fmt.Errorf("could not retrieve categories: %w", err)
	}

	uc.log.Infof("Use Case: Retrieved %d categories", len(categories))
	return categories, nil
}
