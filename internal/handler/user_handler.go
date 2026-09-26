// O handler é responsável por lidar com as requisições HTTP relacionadas aos usuários.
// Ele recebe as requisições, processa os dados de entrada, chama os serviços apropriados e retorna as respostas HTTP.
// O UserHandler é uma estrutura que contém uma referência ao serviço de usuário (UserService) e define os métodos para criar, ler, atualizar e excluir usuários.
// Cada método do UserHandler corresponde a uma rota HTTP específica e utiliza o contexto do Gin para acessar os dados da requisição e enviar as respostas.

package handler

import (
	"errors"
	"gocommerce/internal/model"
	"gocommerce/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService *service.UserService
}

// NewUserHandler cria uma nova instância de UserHandler.
// Parâmetros:
// - userService: ponteiro para o serviço de usuário.
// Retorno:
// - Ponteiro para uma nova instância de UserHandler.
func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// CreateUser cria um novo usuário.
// Parâmetros:
// - c: contexto do Gin que contém os dados da requisição.
// Retorno:
// - JSON com o usuário criado ou mensagem de erro.
func (userhandler *UserHandler) CreateUser(context *gin.Context) {
	// Instancia um novo usuário a partir dos dados da requisição;
	var user model.User

	// Valida os dados de entrada e vincula ao objeto user;
	if err := context.ShouldBindJSON(&user); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Se estiver tudo certo, chama o serviço para criar o usuário;
	created, err := userhandler.userService.CreateUser(&user)
	if err != nil {
		if errors.Is(err, service.ErrEmailAlreadyExists) {
			context.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrEmptyPassword) {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		context.JSON(http.StatusInternalServerError, gin.H{"error": "erro interno do servidor"})
		return
	}

	context.JSON(http.StatusCreated, created)
}

// Login autentica as credenciais do usuário e retorna apenas o token JWT.
// Parâmetros:
// - context: contexto do Gin contendo email e password.
// Retorno:
// - JSON contendo token JWT ou erro de autenticação.
func (userhandler *UserHandler) Login(context *gin.Context) {
	var loginReq model.LoginRequest
	if err := context.ShouldBindJSON(&loginReq); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "dados de login inválidos"})
		return
	}

	token, err := userhandler.userService.Login(loginReq.Email, loginReq.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			context.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		context.JSON(http.StatusInternalServerError, gin.H{"error": "erro interno do servidor"})
		return
	}

	context.JSON(http.StatusOK, model.LoginResponse{
		Token: token,
	})
}

// GetUser retorna um usuário pelo ID.
// Parâmetros:
// - c: contexto do Gin que contém os dados da requisição.
// Retorno:
// - JSON com os dados do usuário ou mensagem de erro.
func (userhandler *UserHandler) GetUser(context *gin.Context) {
	id := context.Param("id")
	user, err := userhandler.userService.GetUserByID(id)
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	context.JSON(http.StatusOK, user)
}

// ListUsers retorna uma lista de todos os usuários.
// Parâmetros:
// - c: contexto do Gin que contém os dados da requisição.
// Retorno:
// - JSON com a lista de usuários ou mensagem de erro.
func (userhandler *UserHandler) ListUsers(context *gin.Context) {
	users, err := userhandler.userService.ListUsers()
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "erro interno do servidor"})
		return
	}

	context.JSON(http.StatusOK, users)
}

// UpdateUser atualiza os dados de um usuário existente.
// Parâmetros:
// - c: contexto do Gin que contém os dados da requisição.
// Retorno:
// - JSON com os dados atualizados do usuário ou mensagem de erro.
func (userhandler *UserHandler) UpdateUser(context *gin.Context) {
	id := context.Param("id")
	user := model.User{ID: id}
	if err := context.ShouldBindJSON(&user); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user.ID = id

	err := userhandler.userService.UpdateUser(&user)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			context.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		context.JSON(http.StatusInternalServerError, gin.H{"error": "erro interno do servidor"})
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

// DeleteUser remove um usuário pelo ID.
// Parâmetros:
// - c: contexto do Gin que contém os dados da requisição.
// Retorno:
// - JSON vazio com status HTTP 204 ou mensagem de erro.
func (userhandler *UserHandler) DeleteUser(context *gin.Context) {
	id := context.Param("id")
	err := userhandler.userService.DeleteUser(id)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			context.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		context.JSON(http.StatusInternalServerError, gin.H{"error": "erro interno do servidor"})
		return
	}

	context.JSON(http.StatusNoContent, nil)
}
