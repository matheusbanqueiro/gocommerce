// O serviço é responsável por implementar a lógica de negócios relacionada aos usuários.
// Ele atua como uma camada intermediária entre os handlers (que lidam com as requisições HTTP) e os repositórios (que interagem com a camada de dados).
// O UserService é uma estrutura que contém uma referência ao UserRepository e define os métodos para criar, ler, atualizar e excluir usuários.
// Ele pode incluir validações, regras de negócios e outras operações relacionadas aos usuários antes de chamar os métodos do repositório.
package service

import (
	"context"
	"errors"
	"gocommerce/internal/model"
	"gocommerce/internal/repository"
	"time"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailAlreadyExists   = errors.New("email já cadastrado")
	ErrEmptyPassword        = errors.New("a senha não pode ser vazia")
	ErrUserNotFound         = errors.New("usuário não encontrado")
	ErrPasswordHash         = errors.New("erro ao criptografar senha")
	ErrInvalidCredentials   = errors.New("credenciais inválidas")
	ErrInvalidOldPassword   = errors.New("senha atual incorreta")
	ErrSamePassword         = errors.New("a nova senha não pode ser igual à senha atual")
	ErrHierarchyViolation   = errors.New("você não tem permissão para gerenciar um usuário com nível hierárquico igual ou superior ao seu")
	ErrDepartmentMismatch   = errors.New("você só tem permissão para gerenciar colaboradores da sua própria área/departamento")
	ErrRoleNotFound         = errors.New("cargo especificado não existe")
	ErrCannotAssignRole     = errors.New("você não pode atribuir um cargo de nível superior ou fora da sua área")
)

type UserService struct {
	userRepo  repository.UserRepository
	jwtSecret string
}

func NewUserService(userRepo repository.UserRepository, jwtSecret string) *UserService {
	return &UserService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

func (s *UserService) InitSchema() error {
	return s.userRepo.InitRBACSchema(context.Background())
}

// CreateUser cria um novo cliente (cadastro público).
func (s *UserService) CreateUser(user *model.User) (*model.User, error) {
	existingUser, err := s.userRepo.GetByEmail(context.Background(), user.Email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrEmailAlreadyExists
	}

	if user.Password == "" {
		return nil, ErrEmptyPassword
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, ErrPasswordHash
	}
	user.Password = string(hashedPassword)
	user.RoleID = "client"
	user.Department = "client"

	err = s.userRepo.Create(context.Background(), user)
	if err != nil {
		return nil, err
	}

	user.Password = ""
	return user, nil
}

// CreateStaff cria um colaborador (estoquista, analista financeiro, etc.) por um gerente da área ou admin.
func (s *UserService) CreateStaff(requesterLevel int, requesterDept string, req *model.CreateStaffRequest) (*model.User, error) {
	targetRole, err := s.userRepo.GetRoleByID(context.Background(), req.RoleID)
	if err != nil {
		return nil, ErrRoleNotFound
	}

	// Regra de escopo do gerente: só cadastra cargos da sua área e de nível menor
	if requesterLevel < 100 {
		if requesterDept != targetRole.Department {
			return nil, ErrDepartmentMismatch
		}
		if requesterLevel <= targetRole.Level {
			return nil, ErrCannotAssignRole
		}
	}

	existingUser, err := s.userRepo.GetByEmail(context.Background(), req.Email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrEmailAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, ErrPasswordHash
	}

	newUser := &model.User{
		Name:       req.Name,
		Email:      req.Email,
		Password:   string(hashedPassword),
		RoleID:     targetRole.ID,
		Department: targetRole.Department,
	}

	if err := s.userRepo.Create(context.Background(), newUser); err != nil {
		return nil, err
	}

	newUser.Password = ""
	return newUser, nil
}

// Login autentica as credenciais do usuário e retorna um token JWT com departamento, nível e permissões.
func (s *UserService) Login(email, password string) (string, error) {
	user, err := s.userRepo.GetByEmail(context.Background(), email)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", ErrInvalidCredentials
	}

	// Gera o token JWT com claims completas de RBAC, departamento e hierarquia
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userID":      user.ID,
		"role":        user.RoleID,
		"department":  user.Department,
		"level":       user.RoleLevel,
		"permissions": user.Permissions,
		"exp":         time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// GetUserByID retorna um usuário pelo ID.
func (s *UserService) GetUserByID(id string) (*model.User, error) {
	user, err := s.userRepo.GetByID(context.Background(), id)
	if err != nil || user == nil {
		return nil, ErrUserNotFound
	}
	user.Password = ""
	return user, nil
}

// ListUsers retorna todos os usuários (Global - Admin).
func (s *UserService) ListUsers() ([]*model.User, error) {
	users, err := s.userRepo.List(context.Background())
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		u.Password = ""
	}
	return users, nil
}

// ListDepartmentUsers retorna os usuários de uma área específica (Gerente da área).
func (s *UserService) ListDepartmentUsers(department string) ([]*model.User, error) {
	users, err := s.userRepo.ListByDepartment(context.Background(), department)
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		u.Password = ""
	}
	return users, nil
}

// UpdateUser atualiza os dados respeitando escopo de departamento e hierarquia.
func (s *UserService) UpdateUser(requesterLevel int, requesterDept, requesterID, targetID string, req *model.UpdateUserRequest) (*model.User, error) {
	targetUser, err := s.userRepo.GetByID(context.Background(), targetID)
	if err != nil || targetUser == nil {
		return nil, ErrUserNotFound
	}

	// Auto-edição é sempre permitida para nome/email
	if requesterID != targetID {
		if requesterLevel < 100 {
			// Gerente só edita quem é da mesma área
			if requesterDept != targetUser.Department {
				return nil, ErrDepartmentMismatch
			}
			// Gerente só edita quem tem nível estritamente menor
			if requesterLevel <= targetUser.RoleLevel {
				return nil, ErrHierarchyViolation
			}
		}
	}

	if req.Name != nil && *req.Name != "" {
		targetUser.Name = *req.Name
	}

	if req.Email != nil && *req.Email != "" && *req.Email != targetUser.Email {
		existing, err := s.userRepo.GetByEmail(context.Background(), *req.Email)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != targetUser.ID {
			return nil, ErrEmailAlreadyExists
		}
		targetUser.Email = *req.Email
	}

	// Alteração de Role
	if req.RoleID != nil && *req.RoleID != "" && *req.RoleID != targetUser.RoleID {
		newRole, err := s.userRepo.GetRoleByID(context.Background(), *req.RoleID)
		if err != nil {
			return nil, ErrRoleNotFound
		}
		if requesterLevel < 100 {
			if requesterDept != newRole.Department || requesterLevel <= newRole.Level {
				return nil, ErrCannotAssignRole
			}
		}
		targetUser.RoleID = newRole.ID
		targetUser.Department = newRole.Department
	}

	if err := s.userRepo.Update(context.Background(), targetUser); err != nil {
		return nil, err
	}

	targetUser.Password = ""
	return targetUser, nil
}

// UpdatePassword altera a senha do usuário após validação da senha atual.
func (s *UserService) UpdatePassword(id string, oldPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(context.Background(), id)
	if err != nil || user == nil {
		return ErrUserNotFound
	}

	if newPassword == "" {
		return ErrEmptyPassword
	}

	if oldPassword == newPassword {
		return ErrSamePassword
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return ErrInvalidOldPassword
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return ErrPasswordHash
	}

	return s.userRepo.UpdatePassword(context.Background(), id, string(hashedPassword))
}

// DeleteUser remove um usuário respeitando escopo de departamento e hierarquia.
func (s *UserService) DeleteUser(requesterLevel int, requesterDept, requesterID, targetID string) error {
	targetUser, err := s.userRepo.GetByID(context.Background(), targetID)
	if err != nil || targetUser == nil {
		return ErrUserNotFound
	}

	if requesterID != targetID {
		if requesterLevel < 100 {
			// Gerente só deleta colaboradores da sua própria área
			if requesterDept != targetUser.Department {
				return ErrDepartmentMismatch
			}
			// Gerente só deleta colaboradores de nível estritamente menor
			if requesterLevel <= targetUser.RoleLevel {
				return ErrHierarchyViolation
			}
		}
	}

	return s.userRepo.Delete(context.Background(), targetID)
}
