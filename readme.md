# 🛒 GoCommerce

API de e-commerce desenvolvida em **Go (Golang)** com foco em performance, simplicidade e boas práticas de arquitetura backend.

---

## 🎯 Objetivo

Construir uma API robusta de e-commerce com:

* Autenticação e autorização via JWT
* Gestão de usuários com identificadores únicos (UUID)
* Controle de permissões (`admin` vs `cliente`)
* Gestão de produtos
* Carrinho de compras
* Processamento de pedidos

---

## ⚙️ Stack Tecnológica

* **Linguagem**: Go (Golang) 1.22+
* **Web Framework**: [Gin Gonic](https://github.com/gin-gonic/gin)
* **Banco de Dados**: PostgreSQL 15
* **Acesso a Dados**: Driver nativo SQL (`database/sql` + `github.com/lib/pq`) — **SQL puro (sem ORMs como GORM)** para máximo desempenho, previsibilidade e controle das queries
* **Autenticação**: JWT (`github.com/dgrijalva/jwt-go`)
* **Migrations**: Scripts versionados em SQL puro (`migrations/`)
* **Containerização**: Docker & Docker Compose

---

## 🧱 Arquitetura do Projeto

Estrutura baseada em camadas com separação clara de responsabilidades:

```
gocommerce/
├── cmd/
│   └── main.go              # Ponto de entrada (setup do servidor, rotas e DI)
├── internal/
│   ├── handler/             # Camada HTTP (recebe request, validação, serialização JSON)
│   ├── middleware/          # Middlewares Gin (autenticação JWT, etc.)
│   ├── model/               # Modelos e entidades de dados
│   ├── repository/          # Acesso ao banco de dados com SQL nativo
│   └── service/             # Regras e lógica de negócio
├── migrations/              # Arquivos versionados de schema do banco (.up.sql e .down.sql)
├── docker-compose.yml       # Configuração dos serviços em container (PostgreSQL)
├── gocommerce.session.sql   # Script SQL para execuções rápidas em IDEs
├── go.mod
└── readme.md
```

### Fluxo de Comunicação:

```mermaid
graph TD
    Client[Cliente / Frontend] -->|Requisição HTTP| Handler[Handler HTTP]
    Handler -->|Valida e chama| Service[Service / Casos de Uso]
    Service -->|Executa lógica e chama| Repository[Repository]
    Repository -->|Executa SQL puro| DB[(PostgreSQL)]
    DB -->|Retorna dados| Repository
    Repository -->|Retorna Model| Service
    Service -->|Retorna resultado| Handler
    Handler -->|Resposta JSON| Client
```

---

## 🐘 Banco de Dados (PostgreSQL)

### 1. Subir o PostgreSQL via Docker

Com o Docker instalado e rodando, execute na raiz do projeto:

```bash
docker compose up -d
```

### 2. Como acessar o PostgreSQL via Terminal (CLI)

Para abrir a interface interativa do `psql` dentro do container:

```bash
docker exec -it gocommerce_postgres psql -U user -d gocommerce
```

*(Senha padrão definida no `docker-compose.yml`: `password`)*

#### 📌 Comandos úteis no `psql`:

| Comando | Descrição |
| :--- | :--- |
| `\dt` | Lista todas as tabelas criadas |
| `\d users` | Descreve colunas, tipos e índices da tabela `users` |
| `SELECT * FROM users;` | Consulta todos os registros de usuários |
| `DELETE FROM users;` | Apaga todos os registros da tabela |
| `\q` | Sai do terminal do PostgreSQL |

### 3. Conexão via Ferramentas Visuais (DBeaver, pgAdmin, VS Code)

* **Host**: `localhost`
* **Porta**: `5432`
* **Database**: `gocommerce`
* **User**: `user`
* **Password**: `password`
* **Connection String (DSN)**:  
  `postgres://user:password@localhost:5432/gocommerce?sslmode=disable`

---

## 🗄️ Migrations

As migrações do banco de dados ficam na pasta [`migrations/`](migrations/):

* `000001_create_users_table.up.sql`: Cria a tabela `users` utilizando UUID nativo (`gen_random_uuid()`).
* `000001_create_users_table.down.sql`: Deleta a tabela `users` caso precise reverter.

### Executar a migration inicial:

#### Opção A — Pelo terminal com Docker:
```bash
docker exec -i gocommerce_postgres psql -U user -d gocommerce < migrations/000001_create_users_table.up.sql
```

#### Opção B — Pelo VS Code / DBeaver:
Abra o arquivo [`migrations/000001_create_users_table.up.sql`](migrations/000001_create_users_table.up.sql) ou [`gocommerce.session.sql`](gocommerce.session.sql) e execute diretamente na sua conexão com o banco.

---

## 🚀 Como Rodar o Projeto

1. **Subir o banco de dados**:
   ```bash
   docker compose up -d
   ```

2. **Aplicar a migration**:
   ```bash
   docker exec -i gocommerce_postgres psql -U user -d gocommerce < migrations/000001_create_users_table.up.sql
   ```

3. **Instalar dependências Go**:
   ```bash
   go mod tidy
   ```

4. **Executar a aplicação**:
   ```bash
   go run cmd/main.go
   ```

O servidor iniciará em **`http://localhost:8080`**.

---

## 🔗 Endpoints da API

### 🔓 Públicos (Sem autenticação)

#### Cadastrar Usuário
* **Método**: `POST`
* **URL**: `http://localhost:8080/auth/register`
* **Body** (`JSON`):
  ```json
  {
    "name": "Matheus Lima",
    "email": "matheus@gmail.com",
    "password": "senha_segura",
    "role": "cliente"
  }
  ```
* **Resposta Sucesso (`201 Created`)**:
  ```json
  {
    "id": "e7b0c39f-93d3-4f9e-a89e-5b128509e5b2",
    "name": "Matheus Lima",
    "email": "matheus@gmail.com",
    "role": "cliente",
    "created_at": "2026-09-26T14:30:00Z",
    "updated_at": "2026-09-26T14:30:00Z"
  }
  ```

#### Login
* **Método**: `POST`
* **URL**: `http://localhost:8080/auth/login`

---

### 🔒 Protegidos (Requer Header `Authorization: Bearer <token>`)

| Método | Rota | Descrição |
| :--- | :--- | :--- |
| `GET` | `/api/users` | Listar todos os usuários |
| `GET` | `/api/users/:id` | Buscar usuário por ID (UUID) |
| `PUT` | `/api/users/:id` | Atualizar dados do usuário |
| `DELETE` | `/api/users/:id` | Excluir usuário |

---

## 📚 Modelos Principais

### User
```json
{
  "id": "uuid",
  "name": "string",
  "email": "string",
  "password": "hash",
  "role": "admin | cliente",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

---

## 📌 Status do Projeto

🚧 **Em desenvolvimento** — Fase atual: módulo de usuários e autenticação com UUID e SQL nativo.

---

## 📄 Licença

MIT
