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
	ErrEmailAlreadyExists = errors.New("email já cadastrado")
	ErrEmptyPassword      = errors.New("a senha não pode ser vazia")
	ErrUserNotFound       = errors.New("usuário não encontrado")
	ErrPasswordHash       = errors.New("erro ao criptografar senha")
	ErrInvalidCredentials = errors.New("credenciais inválidas")
)

type UserService struct {
	userRepo  repository.UserRepository
	jwtSecret string
}

// NewUserService cria uma nova instância de UserService.
// Parâmetros:
// - userRepo: repositório que implementa UserRepository.
// - jwtSecret: chave secreta para assinatura dos tokens JWT.
// Retorno:
// - Ponteiro para a nova instância de UserService.
func NewUserService(userRepo repository.UserRepository, jwtSecret string) *UserService {
	return &UserService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

// CreateUser cria um novo usuário criptografando a senha com hash seguro (bcrypt).
// Parâmetros:
// - user: ponteiro para o modelo de usuário contendo os dados a serem cadastrados.
// Retorno:
// - Ponteiro para o usuário criado (com senha e role omitidas para retorno seguro) ou erro.
func (s *UserService) CreateUser(user *model.User) (*model.User, error) {
	// Verificar se o email já existe
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

	// Gera o hash seguro da senha com salt automático
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, ErrPasswordHash
	}
	user.Password = string(hashedPassword)

	if user.Role == "" {
		user.Role = "client"
	}

	err = s.userRepo.Create(context.Background(), user)
	if err != nil {
		return nil, err
	}

	// Não expõe o password e a role no retorno JSON da API
	user.Password = ""
	user.Role = ""
	return user, nil
}

// Login autentica as credenciais do usuário e retorna um token JWT.
// Parâmetros:
// - email: e-mail fornecido pelo usuário.
// - password: senha em texto plano informada.
// Retorno:
// - Token JWT assinado em string ou erro em caso de falha.
func (s *UserService) Login(email, password string) (string, error) {
	user, err := s.userRepo.GetByEmail(context.Background(), email)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", ErrInvalidCredentials
	}

	// Valida a senha comparando com o hash bcrypt
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", ErrInvalidCredentials
	}

	// Gera o token JWT com claims e expiração de 24 horas
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userID": user.ID,
		"role":   user.Role,
		"exp":    time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// GetUserByID retorna um usuário pelo ID.
// Parâmetros:
// - id: identificador único do usuário (UUID).
// Retorno:
// - Ponteiro para o usuário encontrado ou erro caso não exista.
func (s *UserService) GetUserByID(id string) (*model.User, error) {
	return s.userRepo.GetByID(context.Background(), id)
}

// ListUsers retorna todos os usuários cadastrados.
// Parâmetros:
// - Nenhum.
// Retorno:
// - Slice de ponteiros de usuários ou erro.
func (s *UserService) ListUsers() ([]*model.User, error) {
	return s.userRepo.List(context.Background())
}

// UpdateUser atualiza os dados de um usuário existente.
// Parâmetros:
// - user: ponteiro para o modelo de usuário contendo o ID e os novos dados.
// Retorno:
// - Erro em caso de falha ou nil em caso de sucesso.
func (s *UserService) UpdateUser(user *model.User) error {
	_, err := s.userRepo.GetByID(context.Background(), user.ID)
	if err != nil {
		return ErrUserNotFound
	}

	if user.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			return ErrPasswordHash
		}
		user.Password = string(hashedPassword)
	}

	return s.userRepo.Update(context.Background(), user)
}

// DeleteUser remove um usuário pelo ID.
// Parâmetros:
// - id: identificador único do usuário a ser removido.
// Retorno:
// - Erro em caso de falha ou nil em caso de sucesso.
func (s *UserService) DeleteUser(id string) error {
	_, err := s.userRepo.GetByID(context.Background(), id)
	if err != nil {
		return ErrUserNotFound
	}
	return s.userRepo.Delete(context.Background(), id)
}
