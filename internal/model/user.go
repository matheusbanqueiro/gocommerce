package model

import "time"

// Role representa a estrutura de um cargo na hierarquia associado a um departamento.
type Role struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Department  string    `json:"department"` // 'system', 'finance', 'stock', 'client'
	Description string    `json:"description,omitempty"`
	Level       int       `json:"level"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Permission representa uma permissão atômica do sistema.
type Permission struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Department  string    `json:"department"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// User representa a entidade de usuário com informações de RBAC, departamento e hierarquia.
type User struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Password    string    `json:"password,omitempty"`
	RoleID      string    `json:"role_id,omitempty"`
	RoleName    string    `json:"role_name,omitempty"`
	Department  string    `json:"department,omitempty"`
	RoleLevel   int       `json:"role_level,omitempty"`
	Permissions []string  `json:"permissions,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// LoginRequest representa a estrutura dos dados recebidos no login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse representa o retorno da autenticação contendo o token JWT.
type LoginResponse struct {
	Token string `json:"token"`
}

// UpdateUserRequest representa os dados permitidos para atualização parcial de usuário.
type UpdateUserRequest struct {
	Name   *string `json:"name,omitempty"`
	Email  *string `json:"email,omitempty"`
	RoleID *string `json:"role_id,omitempty"`
}

// UpdatePasswordRequest representa a estrutura para alteração de senha.
type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// CreateStaffRequest representa a criação de colaboradores por gerentes ou administradores.
type CreateStaffRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	RoleID   string `json:"role_id" binding:"required"`
}
