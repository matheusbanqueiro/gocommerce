// Os repositórios são responsáveis por interagir com a camada de dados, como bancos de dados ou APIs externas.
// Eles fornecem métodos para criar, ler, atualizar e excluir dados relacionados aos usuários.
// O UserRepository é uma interface que define os métodos necessários para gerenciar os usuários,
// enquanto a implementação concreta (userRepository) utiliza um banco de dados SQL para armazenar e recuperar os dados dos usuários.

package repository

import (
	"context"
	"database/sql"
	"errors"
	"gocommerce/internal/model"

	"github.com/lib/pq"
)

type UserRepository interface {
	InitRBACSchema(ctx context.Context) error
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetRoleByID(ctx context.Context, roleID string) (*model.Role, error)
	Update(ctx context.Context, user *model.User) error
	UpdatePassword(ctx context.Context, id, hashedPassword string) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*model.User, error)
	ListByDepartment(ctx context.Context, department string) ([]*model.User, error)
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

// InitRBACSchema inicializa as tabelas de roles, departamentos, permissões e seeds de dados.
func (r *userRepository) InitRBACSchema(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS roles (
		id VARCHAR(50) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		department VARCHAR(50) NOT NULL DEFAULT 'client',
		description TEXT,
		level INT NOT NULL DEFAULT 1,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS permissions (
		id VARCHAR(100) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		department VARCHAR(50) NOT NULL DEFAULT 'system',
		description TEXT,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS role_permissions (
		role_id VARCHAR(50) NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
		permission_id VARCHAR(100) NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
		PRIMARY KEY (role_id, permission_id)
	);

	ALTER TABLE users ADD COLUMN IF NOT EXISTS role_id VARCHAR(50) DEFAULT 'client';
	ALTER TABLE users ADD COLUMN IF NOT EXISTS department VARCHAR(50) DEFAULT 'client';

	INSERT INTO roles (id, name, department, description, level) VALUES
		('admin', 'Administrador Geral', 'system', 'Acesso global irrestrito a todas as áreas', 100),
		('finance_manager', 'Gerente Financeiro', 'finance', 'Gerencia custos, relatórios e equipe do financeiro', 50),
		('finance_analyst', 'Analista Financeiro', 'finance', 'Operações e consultas financeiras', 20),
		('stock_manager', 'Gerente de Estoque', 'stock', 'Gerencia catálogo, inventário e equipe de estoquistas', 50),
		('stockist', 'Estoquista', 'stock', 'Cadastro e atualização de produtos e estoque', 20),
		('client', 'Cliente', 'client', 'Cliente consumidor da plataforma', 1)
	ON CONFLICT (id) DO UPDATE SET level = EXCLUDED.level, name = EXCLUDED.name, department = EXCLUDED.department;

	INSERT INTO permissions (id, name, department, description) VALUES
		('users:read_all', 'Listar Todos Usuários', 'system', 'Visualizar usuários de todas as áreas'),
		('users:write_all', 'Editar Qualquer Usuário', 'system', 'Editar usuários de qualquer departamento'),
		('users:delete_all', 'Deletar Qualquer Usuário', 'system', 'Deletar usuários de qualquer área (nível inferior)'),
		('roles:manage', 'Gerenciar Cargos e Permissões', 'system', 'Criar e configurar roles'),

		('dept_users:read', 'Listar Usuários do Departamento', 'system', 'Ver colaboradores da sua própria área'),
		('dept_users:create', 'Cadastrar na sua Área', 'system', 'Contratar/cadastrar funcionários na sua área'),
		('dept_users:write', 'Editar Colaborador da Área', 'system', 'Editar dados de subordinados da sua área'),
		('dept_users:delete', 'Demitir/Deletar da sua Área', 'system', 'Deletar subordinados da sua própria área'),

		('products:read', 'Visualizar Produtos', 'stock', 'Consultar catálogo de produtos'),
		('products:create', 'Cadastrar Produtos', 'stock', 'Adicionar novos produtos no catálogo'),
		('products:write', 'Editar Produtos', 'stock', 'Atualizar preços, fotos e descrições'),
		('products:delete', 'Deletar Produtos', 'stock', 'Remover produtos do catálogo'),
		('stock:manage', 'Ajustar Inventário', 'stock', 'Dar entrada/saída em quantidades de estoque'),

		('finance:read', 'Visualizar Custos e Entradas', 'finance', 'Consultar relatórios financeiros e custos'),
		('finance:write', 'Gerenciar Lançamentos Financeiros', 'finance', 'Criar e editar contas a pagar/receber'),
		('finance:reports', 'Exportar Balanços Financeiros', 'finance', 'Gerar relatórios de faturamento e lucro'),

		('users:self:read', 'Ver Próprio Perfil', 'client', 'Visualizar seus próprios dados'),
		('users:self:write', 'Editar Próprio Perfil', 'client', 'Alterar seus dados cadastrais e senha'),
		('users:self:delete', 'Excluir Própria Conta', 'client', 'Encerrar sua própria conta')
	ON CONFLICT (id) DO NOTHING;

	-- Admin
	INSERT INTO role_permissions (role_id, permission_id)
	SELECT 'admin', id FROM permissions
	ON CONFLICT DO NOTHING;

	-- Stock Manager
	INSERT INTO role_permissions (role_id, permission_id) VALUES
		('stock_manager', 'dept_users:read'),
		('stock_manager', 'dept_users:create'),
		('stock_manager', 'dept_users:write'),
		('stock_manager', 'dept_users:delete'),
		('stock_manager', 'products:read'),
		('stock_manager', 'products:create'),
		('stock_manager', 'products:write'),
		('stock_manager', 'products:delete'),
		('stock_manager', 'stock:manage'),
		('stock_manager', 'users:self:read'),
		('stock_manager', 'users:self:write'),
		('stock_manager', 'users:self:delete')
	ON CONFLICT DO NOTHING;

	-- Stockist
	INSERT INTO role_permissions (role_id, permission_id) VALUES
		('stockist', 'products:read'),
		('stockist', 'products:create'),
		('stockist', 'products:write'),
		('stockist', 'stock:manage'),
		('stockist', 'users:self:read'),
		('stockist', 'users:self:write'),
		('stockist', 'users:self:delete')
	ON CONFLICT DO NOTHING;

	-- Finance Manager
	INSERT INTO role_permissions (role_id, permission_id) VALUES
		('finance_manager', 'dept_users:read'),
		('finance_manager', 'dept_users:create'),
		('finance_manager', 'dept_users:write'),
		('finance_manager', 'dept_users:delete'),
		('finance_manager', 'finance:read'),
		('finance_manager', 'finance:write'),
		('finance_manager', 'finance:reports'),
		('finance_manager', 'users:self:read'),
		('finance_manager', 'users:self:write'),
		('finance_manager', 'users:self:delete')
	ON CONFLICT DO NOTHING;

	-- Finance Analyst
	INSERT INTO role_permissions (role_id, permission_id) VALUES
		('finance_analyst', 'finance:read'),
		('finance_analyst', 'finance:write'),
		('finance_analyst', 'users:self:read'),
		('finance_analyst', 'users:self:write'),
		('finance_analyst', 'users:self:delete')
	ON CONFLICT DO NOTHING;

	-- Client
	INSERT INTO role_permissions (role_id, permission_id) VALUES
		('client', 'products:read'),
		('client', 'users:self:read'),
		('client', 'users:self:write'),
		('client', 'users:self:delete')
	ON CONFLICT DO NOTHING;
	`
	_, err := r.db.ExecContext(ctx, query)
	return err
}

func (r *userRepository) GetRoleByID(ctx context.Context, roleID string) (*model.Role, error) {
	query := `SELECT id, name, department, COALESCE(description, ''), level, created_at, updated_at FROM roles WHERE id = $1`
	role := &model.Role{}
	err := r.db.QueryRowContext(ctx, query, roleID).Scan(&role.ID, &role.Name, &role.Department, &role.Description, &role.Level, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("cargo não encontrado")
		}
		return nil, err
	}
	return role, nil
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	if user.RoleID == "" {
		user.RoleID = "client"
	}
	if user.Department == "" {
		user.Department = "client"
	}

	query := `
	INSERT INTO users (name, email, password, role_id, department) 
	VALUES ($1, $2, $3, $4, $5) 
	RETURNING id, created_at, updated_at`
	
	return r.db.QueryRowContext(ctx, query, user.Name, user.Email, user.Password, user.RoleID, user.Department).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	query := `
	SELECT 
		u.id, u.name, u.email, u.password, u.role_id, COALESCE(r.name, ''), COALESCE(r.department, 'client'), COALESCE(r.level, 1),
		COALESCE(ARRAY_AGG(rp.permission_id) FILTER (WHERE rp.permission_id IS NOT NULL), '{}') AS permissions,
		u.created_at, u.updated_at
	FROM users u
	LEFT JOIN roles r ON u.role_id = r.id
	LEFT JOIN role_permissions rp ON r.id = rp.role_id
	WHERE u.id = $1
	GROUP BY u.id, u.name, u.email, u.password, u.role_id, r.name, r.department, r.level, u.created_at, u.updated_at`

	user := &model.User{}
	var permissions []string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Name, &user.Email, &user.Password,
		&user.RoleID, &user.RoleName, &user.Department, &user.RoleLevel,
		pq.Array(&permissions),
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("usuário não encontrado")
		}
		return nil, err
	}
	user.Permissions = permissions
	return user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
	SELECT 
		u.id, u.name, u.email, u.password, u.role_id, COALESCE(r.name, ''), COALESCE(r.department, 'client'), COALESCE(r.level, 1),
		COALESCE(ARRAY_AGG(rp.permission_id) FILTER (WHERE rp.permission_id IS NOT NULL), '{}') AS permissions,
		u.created_at, u.updated_at
	FROM users u
	LEFT JOIN roles r ON u.role_id = r.id
	LEFT JOIN role_permissions rp ON r.id = rp.role_id
	WHERE u.email = $1
	GROUP BY u.id, u.name, u.email, u.password, u.role_id, r.name, r.department, r.level, u.created_at, u.updated_at`

	user := &model.User{}
	var permissions []string

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Name, &user.Email, &user.Password,
		&user.RoleID, &user.RoleName, &user.Department, &user.RoleLevel,
		pq.Array(&permissions),
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	user.Permissions = permissions
	return user, nil
}

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	query := `UPDATE users SET name = $1, email = $2, role_id = $3, department = $4, updated_at = NOW() WHERE id = $5 RETURNING updated_at`
	return r.db.QueryRowContext(ctx, query, user.Name, user.Email, user.RoleID, user.Department, user.ID).Scan(&user.UpdatedAt)
}

func (r *userRepository) UpdatePassword(ctx context.Context, id, hashedPassword string) error {
	query := `UPDATE users SET password = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, hashedPassword, id)
	return err
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *userRepository) List(ctx context.Context) ([]*model.User, error) {
	query := `
	SELECT 
		u.id, u.name, u.email, u.role_id, COALESCE(r.name, ''), COALESCE(r.department, 'client'), COALESCE(r.level, 1),
		u.created_at, u.updated_at
	FROM users u
	LEFT JOIN roles r ON u.role_id = r.id
	ORDER BY r.level DESC, u.name ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		user := &model.User{}
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.RoleID, &user.RoleName, &user.Department, &user.RoleLevel, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *userRepository) ListByDepartment(ctx context.Context, department string) ([]*model.User, error) {
	query := `
	SELECT 
		u.id, u.name, u.email, u.role_id, COALESCE(r.name, ''), COALESCE(r.department, 'client'), COALESCE(r.level, 1),
		u.created_at, u.updated_at
	FROM users u
	LEFT JOIN roles r ON u.role_id = r.id
	WHERE r.department = $1
	ORDER BY r.level DESC, u.name ASC`

	rows, err := r.db.QueryContext(ctx, query, department)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		user := &model.User{}
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.RoleID, &user.RoleName, &user.Department, &user.RoleLevel, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}
