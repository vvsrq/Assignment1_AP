package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"user_service/internal/domain"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type postgresUserRepository struct {
	db  *sql.DB
	log *logrus.Logger
}

func NewPostgresUserRepository(db *sql.DB, logger *logrus.Logger) domain.UserRepository {
	return &postgresUserRepository{
		db:  db,
		log: logger,
	}
}

func (r *postgresUserRepository) CreateUser(user *domain.User) (*domain.User, error) {
	query := `
        INSERT INTO users (name, email, password_hash)
        VALUES ($1, $2, $3)
        RETURNING id, created_at, updated_at`

	r.log.Debugf("Repository: Attempting to create user with email: %s", user.Email)

	err := r.db.QueryRow(query, user.Name, user.Email, user.PasswordHash).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {

		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			r.log.Warnf("Repository: Attempted to create user with duplicate email: %s", user.Email)
			return nil, fmt.Errorf("user with email '%s' already exists", user.Email)
		}

		r.log.Errorf("Repository: Failed to create user '%s': %v", user.Email, err)
		return nil, fmt.Errorf("could not create user: %w", err)
	}

	r.log.Infof("Repository: User created successfully with ID: %d, Email: %s", user.ID, user.Email)
	return user, nil
}

func (r *postgresUserRepository) GetUserByEmail(email string) (*domain.User, error) {
	query := `
        SELECT id, name, email, password_hash, created_at, updated_at
        FROM users
        WHERE email = $1`
	user := &domain.User{}

	r.log.Debugf("Repository: Attempting to find user by email: %s", email)

	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.Warnf("Repository: User with email %s not found", email)

			return nil, fmt.Errorf("user with email %s not found", email)
		}
		r.log.Errorf("Repository: Failed to get user by email %s: %v", email, err)
		return nil, fmt.Errorf("could not get user by email: %w", err)
	}

	r.log.Debugf("Repository: User found by email %s (ID: %d)", email, user.ID)
	return user, nil
}

func (r *postgresUserRepository) GetUserByID(id int64) (*domain.User, error) {
	query := `
        SELECT id, name, email, password_hash, created_at, updated_at
        FROM users
        WHERE id = $1`
	user := &domain.User{}

	r.log.Debugf("Repository: Attempting to find user by ID: %d", id)

	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.Warnf("Repository: User with ID %d not found", id)

			return nil, fmt.Errorf("user with id %d not found", id)
		}
		r.log.Errorf("Repository: Failed to get user by ID %d: %v", id, err)
		return nil, fmt.Errorf("could not get user by id: %w", err)
	}

	r.log.Debugf("Repository: User found by ID %d (Email: %s)", id, user.Email)
	return user, nil
}
