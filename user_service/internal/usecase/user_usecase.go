package usecase

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"user_service/internal/domain" // Убедись, что путь импорта правильный

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// userUseCase implements the domain.UserUseCase interface
type userUseCase struct {
	userRepo domain.UserRepository
	log      *logrus.Logger
	// Можно добавить сюда секрет для JWT, если будем генерировать его здесь
}

// NewUserUseCase creates a new instance of userUseCase
func NewUserUseCase(repo domain.UserRepository, logger *logrus.Logger) domain.UserUseCase {
	return &userUseCase{
		userRepo: repo,
		log:      logger,
	}
}

// RegisterUser handles user registration, including validation and password hashing
func (uc *userUseCase) RegisterUser(name, email, password string) (*domain.User, error) {
	uc.log.Infof("Use Case: Attempting registration for email: %s", email)

	// 1. Basic Validation
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email)) // Normalize email

	if name == "" {
		uc.log.Warn("Use Case: Registration failed - empty name")
		return nil, errors.New("user name cannot be empty")
	}
	if !isValidEmail(email) { // Простая проверка email
		uc.log.Warnf("Use Case: Registration failed - invalid email format: %s", email)
		return nil, errors.New("invalid email format")
	}
	if err := validatePassword(password); err != nil { // Проверка сложности пароля
		uc.log.Warnf("Use Case: Registration failed - password validation error: %v", err)
		return nil, err
	}

	// 2. Check if email already exists
	// Мы полагаемся на ошибку уникальности от репозитория, но можно и явно проверить
	// _, err := uc.userRepo.GetUserByEmail(email)
	// if err == nil {
	//  uc.log.Warnf("Use Case: Registration failed - email already exists: %s", email)
	//  return nil, fmt.Errorf("user with email '%s' already exists", email)
	// }
	// if !errors.Is(err, sql.ErrNoRows) && !strings.Contains(err.Error(), "not found") { // Проверяем, что ошибка именно "не найдено"
	//  // Если другая ошибка (например, DB недоступна), пробрасываем ее
	//  uc.log.Errorf("Use Case: Error checking email existence for %s: %v", email, err)
	//  return nil, fmt.Errorf("failed to check email existence: %w", err)
	// }
	// Оставим проверку на уровне репозитория (unique constraint) для атомарности

	// 3. Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		uc.log.Errorf("Use Case: Failed to hash password for %s: %v", email, err)
		return nil, fmt.Errorf("internal error processing password: %w", err)
	}

	// 4. Create User object
	newUser := &domain.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hashedPassword),
	}

	// 5. Save to repository
	createdUser, err := uc.userRepo.CreateUser(newUser)
	if err != nil {
		uc.log.Errorf("Use Case: Repository failed to create user %s: %v", email, err)
		// Возвращаем ошибку как есть (она уже содержит сообщение про duplicate email)
		return nil, err
	}

	uc.log.Infof("Use Case: User registered successfully. ID: %d, Email: %s", createdUser.ID, createdUser.Email)
	// Возвращаем пользователя без хеша пароля (если нужно для ответа gRPC)
	// Хотя RegisterUser в proto возвращает User (который без хеша),
	// здесь можно вернуть полного пользователя, а в gRPC хендлере отфильтровать.
	return createdUser, nil
}

// AuthenticateUser handles user login
func (uc *userUseCase) AuthenticateUser(email, password string) (*domain.AuthResponse, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	uc.log.Infof("Use Case: Attempting authentication for email: %s", email)

	if !isValidEmail(email) || password == "" {
		uc.log.Warnf("Use Case: Auth failed - invalid email or empty password for %s", email)
		return &domain.AuthResponse{Authenticated: false, ErrorMessage: "Invalid email or password"}, nil // Не ошибка, а результат "не аутентифицирован"
	}

	// 1. Get user by email
	user, err := uc.userRepo.GetUserByEmail(email)
	if err != nil {
		// Если пользователь не найден
		if strings.Contains(err.Error(), "not found") {
			uc.log.Warnf("Use Case: Auth failed - user not found: %s", email)
			return &domain.AuthResponse{Authenticated: false, ErrorMessage: "Invalid email or password"}, nil
		}
		// Если другая ошибка БД
		uc.log.Errorf("Use Case: Error retrieving user %s during auth: %v", email, err)
		return nil, fmt.Errorf("failed to retrieve user: %w", err) // Это внутренняя ошибка
	}

	// 2. Compare the provided password with the stored hash
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		// Если пароли не совпадают (bcrypt.ErrMismatchedHashAndPassword)
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			uc.log.Warnf("Use Case: Auth failed - incorrect password for user %s (ID: %d)", email, user.ID)
			return &domain.AuthResponse{Authenticated: false, ErrorMessage: "Invalid email or password"}, nil
		}
		// Если другая ошибка при сравнении (маловероятно)
		uc.log.Errorf("Use Case: Error comparing password hash for user %s: %v", email, err)
		return nil, fmt.Errorf("internal error during authentication: %w", err) // Внутренняя ошибка
	}

	// 3. Authentication successful - Generate a simple token (UUID for now)
	token := uuid.NewString()
	uc.log.Infof("Use Case: Authentication successful for user %s (ID: %d). Generated token: %s", email, user.ID, token)

	return &domain.AuthResponse{
		Authenticated: true,
		Token:         token,
		UserID:        user.ID,
	}, nil
}

// GetUserProfile retrieves user profile information
func (uc *userUseCase) GetUserProfile(id int64) (*domain.UserProfile, error) {
	uc.log.Infof("Use Case: Attempting to get profile for user ID: %d", id)

	if id <= 0 {
		uc.log.Warnf("Use Case: Get profile failed - invalid user ID: %d", id)
		return nil, errors.New("invalid user ID")
	}

	user, err := uc.userRepo.GetUserByID(id)
	if err != nil {
		uc.log.Warnf("Use Case: Repository failed to get user profile for ID %d: %v", id, err)
		// Возвращаем ошибку как есть (включая not found)
		return nil, err
	}

	// Map domain.User to domain.UserProfile (excluding sensitive info like hash)
	profile := &domain.UserProfile{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
	}

	uc.log.Infof("Use Case: Profile retrieved successfully for user ID: %d", id)
	return profile, nil
}

// --- Helper Functions ---

// isValidEmail provides a basic check for email format.
// For production, consider a more robust library.
func isValidEmail(email string) bool {
	// Простейшая проверка: есть '@', есть '.' после '@', не пустые части
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	domainParts := strings.Split(parts[1], ".")
	return len(domainParts) >= 2 && domainParts[0] != "" && domainParts[len(domainParts)-1] != ""
}

// validatePassword enforces basic password complexity rules.
func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	hasUpper := false
	hasLower := false
	hasDigit := false
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		}
	}
	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return errors.New("password must contain at least one digit")
	}
	return nil
}
