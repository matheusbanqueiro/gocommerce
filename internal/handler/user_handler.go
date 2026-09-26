// O handler é responsável por lidar com as requisições HTTP relacionadas aos usuários.
// Ele recebe as requisições, processa os dados de entrada, chama os serviços apropriados e retorna as respostas HTTP.
// O UserHandler é uma estrutura que contém uma referência ao serviço de usuário (UserService) e define os métodos para criar, ler, atualizar e excluir usuários.
// Cada método do UserHandler corresponde a uma rota HTTP específica e utiliza o contexto do Gin para acessar os dados da requisição e enviar as respostas.

package handler

import (
	"errors"
	"gocommerce/internal/middleware"
	"gocommerce/internal/model"
	"gocommerce/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// CreateUser cria um novo cliente (cadastro aberto ao público).
func (userhandler *UserHandler) CreateUser(context *gin.Context) {
	var user model.User

	if err := context.ShouldBindJSON(&user); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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

// CreateStaff permite que gerentes da área ou administradores cadastrem novos colaboradores.
func (userhandler *UserHandler) CreateStaff(context *gin.Context) {
	var req model.CreateStaffRequest
	if err := context.ShouldBindJSON(&req); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "dados de cadastro de colaborador inválidos"})
		return
	}

	requesterLevel := 1
	if lvl, exists := context.Get("level"); exists {
		if l, ok := lvl.(int); ok {
			requesterLevel = l
		}
	}
	requesterDept := context.GetString("department")

	created, err := userhandler.userService.CreateStaff(requesterLevel, requesterDept, &req)
	if err != nil {
		if errors.Is(err, service.ErrEmailAlreadyExists) {
			context.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrRoleNotFound) || errors.Is(err, service.ErrCannotAssignRole) || errors.Is(err, service.ErrDepartmentMismatch) {
			context.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		context.JSON(http.StatusInternalServerError, gin.H{"error": "erro interno do servidor"})
		return
	}

	context.JSON(http.StatusCreated, created)
}

// Login autentica o usuário e retorna o token JWT com claims de área e permissões.
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

// GetProfile retorna os dados do usuário autenticado no token JWT.
func (userhandler *UserHandler) GetProfile(context *gin.Context) {
	userID := context.GetString("userID")
	user, err := userhandler.userService.GetUserByID(userID)
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"error": "usuário não encontrado"})
		return
	}

	context.JSON(http.StatusOK, user)
}

// GetUser retorna um usuário por ID.
func (userhandler *UserHandler) GetUser(context *gin.Context) {
	id := context.Param("id")
	user, err := userhandler.userService.GetUserByID(id)
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"error": "usuário não encontrado"})
		return
	}

	context.JSON(http.StatusOK, user)
}

// ListUsers lista todos os usuários de todo o sistema (Admin).
func (userhandler *UserHandler) ListUsers(context *gin.Context) {
	users, err := userhandler.userService.ListUsers()
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "erro interno do servidor"})
		return
	}

	context.JSON(http.StatusOK, users)
}

// ListDepartmentUsers lista colaboradores da área do gerente autenticado.
func (userhandler *UserHandler) ListDepartmentUsers(context *gin.Context) {
	department := context.GetString("department")
	users, err := userhandler.userService.ListDepartmentUsers(department)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "erro interno do servidor"})
		return
	}

	context.JSON(http.StatusOK, users)
}

// UpdateUser atualiza dados respeitando hierarquia e departamento.
func (userhandler *UserHandler) UpdateUser(context *gin.Context) {
	targetID := context.Param("id")
	requesterID := context.GetString("userID")
	requesterDept := context.GetString("department")
	requesterLevel := 1
	if lvl, exists := context.Get("level"); exists {
		if l, ok := lvl.(int); ok {
			requesterLevel = l
		}
	}

	var req model.UpdateUserRequest
	if err := context.ShouldBindJSON(&req); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "dados de atualização inválidos"})
		return
	}

	updatedUser, err := userhandler.userService.UpdateUser(requesterLevel, requesterDept, requesterID, targetID, &req)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			context.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrEmailAlreadyExists) {
			context.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrHierarchyViolation) || errors.Is(err, service.ErrDepartmentMismatch) || errors.Is(err, service.ErrCannotAssignRole) {
			context.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		context.JSON(http.StatusInternalServerError, gin.H{"error": "erro interno do servidor"})
		return
	}

	context.JSON(http.StatusOK, updatedUser)
}

// UpdatePassword altera a senha com validação da senha antiga.
func (userhandler *UserHandler) UpdatePassword(context *gin.Context) {
	targetID := context.Param("id")
	requesterID := context.GetString("userID")

	if targetID != requesterID {
		context.JSON(http.StatusForbidden, gin.H{"error": "você só pode alterar a senha da sua própria conta"})
		return
	}

	var req model.UpdatePasswordRequest
	if err := context.ShouldBindJSON(&req); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "dados de troca de senha inválidos"})
		return
	}

	err := userhandler.userService.UpdatePassword(targetID, req.OldPassword, req.NewPassword)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			context.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrInvalidOldPassword) || errors.Is(err, service.ErrSamePassword) || errors.Is(err, service.ErrEmptyPassword) {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		context.JSON(http.StatusInternalServerError, gin.H{"error": "erro interno do servidor"})
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "senha alterada com sucesso"})
}

// DeleteUser remove um usuário respeitando o departamento e o nível hierárquico.
func (userhandler *UserHandler) DeleteUser(context *gin.Context) {
	targetID := context.Param("id")
	requesterID := context.GetString("userID")
	requesterDept := context.GetString("department")
	requesterLevel := 1
	if lvl, exists := context.Get("level"); exists {
		if l, ok := lvl.(int); ok {
			requesterLevel = l
		}
	}

	err := userhandler.userService.DeleteUser(requesterLevel, requesterDept, requesterID, targetID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			context.JSON(http.StatusNotFound, gin.H{"error": "usuário não encontrado"})
			return
		}
		if errors.Is(err, service.ErrHierarchyViolation) || errors.Is(err, service.ErrDepartmentMismatch) {
			context.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		context.JSON(http.StatusInternalServerError, gin.H{"error": "erro interno do servidor"})
		return
	}

	if targetID == requesterID {
		rawToken := context.GetString("rawToken")
		if rawToken != "" {
			middleware.InvalidateToken(rawToken)
		}
	}

	context.JSON(http.StatusOK, gin.H{"message": "usuário excluído com sucesso"})
}

// DeleteOwnAccount encerra a própria conta do usuário autenticado.
func (userhandler *UserHandler) DeleteOwnAccount(context *gin.Context) {
	userID := context.GetString("userID")
	err := userhandler.userService.DeleteUser(9999, "system", userID, userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			context.JSON(http.StatusNotFound, gin.H{"error": "usuário não encontrado"})
			return
		}
		context.JSON(http.StatusInternalServerError, gin.H{"error": "erro interno do servidor"})
		return
	}

	rawToken := context.GetString("rawToken")
	if rawToken != "" {
		middleware.InvalidateToken(rawToken)
	}

	context.JSON(http.StatusOK, gin.H{"message": "sua conta foi excluída com sucesso e seu token foi invalidado"})
}
