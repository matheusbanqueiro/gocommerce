-- Criação das tabelas para RBAC, Departamentos e Hierarquia

-- 1. Tabela de Cargos/Roles
CREATE TABLE IF NOT EXISTS roles (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    department VARCHAR(50) NOT NULL DEFAULT 'client', -- 'system', 'finance', 'stock', 'client'
    description TEXT,
    level INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Tabela de Permissões
CREATE TABLE IF NOT EXISTS permissions (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    department VARCHAR(50) NOT NULL DEFAULT 'system',
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. Tabela associativa Role <-> Permissions
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id VARCHAR(50) NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id VARCHAR(100) NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- 4. Ajustes na tabela users
ALTER TABLE users ADD COLUMN IF NOT EXISTS role_id VARCHAR(50) DEFAULT 'client';
ALTER TABLE users ADD COLUMN IF NOT EXISTS department VARCHAR(50) DEFAULT 'client';

-- Seed de Cargos Padrão com Departamentos e Níveis
INSERT INTO roles (id, name, department, description, level) VALUES
    ('admin', 'Administrador Geral', 'system', 'Acesso global irrestrito a todas as áreas', 100),
    ('finance_manager', 'Gerente Financeiro', 'finance', 'Gerencia custos, relatórios e equipe do financeiro', 50),
    ('finance_analyst', 'Analista Financeiro', 'finance', 'Operações e consultas financeiras', 20),
    ('stock_manager', 'Gerente de Estoque', 'stock', 'Gerencia catálogo, inventário e equipe de estoquistas', 50),
    ('stockist', 'Estoquista', 'stock', 'Cadastro e atualização de produtos e estoque', 20),
    ('client', 'Cliente', 'client', 'Cliente consumidor da plataforma', 1)
ON CONFLICT (id) DO UPDATE SET level = EXCLUDED.level, name = EXCLUDED.name, department = EXCLUDED.department;

-- Seed de Permissões por Departamento
INSERT INTO permissions (id, name, department, description) VALUES
    -- Permissões Globais / Admin
    ('users:read_all', 'Listar Todos Usuários', 'system', 'Visualizar usuários de todas as áreas'),
    ('users:write_all', 'Editar Qualquer Usuário', 'system', 'Editar usuários de qualquer departamento'),
    ('users:delete_all', 'Deletar Qualquer Usuário', 'system', 'Deletar usuários de qualquer área (nível inferior)'),
    ('roles:manage', 'Gerenciar Cargos e Permissões', 'system', 'Criar e configurar roles'),

    -- Permissões Departamentais (Gerentes)
    ('dept_users:read', 'Listar Usuários do Departamento', 'system', 'Ver colaboradores da sua própria área'),
    ('dept_users:create', 'Cadastrar na sua Área', 'system', 'Contratar/cadastrar funcionários na sua área'),
    ('dept_users:write', 'Editar Colaborador da Área', 'system', 'Editar dados de subordinados da sua área'),
    ('dept_users:delete', 'Demitir/Deletar da sua Área', 'system', 'Deletar subordinados da sua própria área'),

    -- Permissões do Estoque / Produtos
    ('products:read', 'Visualizar Produtos', 'stock', 'Consultar catálogo de produtos'),
    ('products:create', 'Cadastrar Produtos', 'stock', 'Adicionar novos produtos no catálogo'),
    ('products:write', 'Editar Produtos', 'stock', 'Atualizar preços, fotos e descrições'),
    ('products:delete', 'Deletar Produtos', 'stock', 'Remover produtos do catálogo'),
    ('stock:manage', 'Ajustar Inventário', 'stock', 'Dar entrada/saída em quantidades de estoque'),

    -- Permissões do Financeiro
    ('finance:read', 'Visualizar Custos e Entradas', 'finance', 'Consultar relatórios financeiros e custos'),
    ('finance:write', 'Gerenciar Lançamentos Financeiros', 'finance', 'Criar e editar contas a pagar/receber'),
    ('finance:reports', 'Exportar Balanços Financeiros', 'finance', 'Gerar relatórios de faturamento e lucro'),

    -- Permissões de Auto-gestão (Qualquer usuário)
    ('users:self:read', 'Ver Próprio Perfil', 'client', 'Visualizar seus próprios dados'),
    ('users:self:write', 'Editar Próprio Perfil', 'client', 'Alterar seus dados cadastrais e senha'),
    ('users:self:delete', 'Excluir Própria Conta', 'client', 'Encerrar sua própria conta')
ON CONFLICT (id) DO NOTHING;

-- Associação de Permissões:

-- 1. Admin (Possui todas as permissões)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 'admin', id FROM permissions
ON CONFLICT DO NOTHING;

-- 2. Gerente de Estoque (stock_manager)
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

-- 3. Estoquista (stockist)
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('stockist', 'products:read'),
    ('stockist', 'products:create'),
    ('stockist', 'products:write'),
    ('stockist', 'stock:manage'),
    ('stockist', 'users:self:read'),
    ('stockist', 'users:self:write'),
    ('stockist', 'users:self:delete')
ON CONFLICT DO NOTHING;

-- 4. Gerente Financeiro (finance_manager)
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

-- 5. Analista Financeiro (finance_analyst)
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('finance_analyst', 'finance:read'),
    ('finance_analyst', 'finance:write'),
    ('finance_analyst', 'users:self:read'),
    ('finance_analyst', 'users:self:write'),
    ('finance_analyst', 'users:self:delete')
ON CONFLICT DO NOTHING;

-- 6. Cliente (client)
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('client', 'products:read'),
    ('client', 'users:self:read'),
    ('client', 'users:self:write'),
    ('client', 'users:self:delete')
ON CONFLICT DO NOTHING;
