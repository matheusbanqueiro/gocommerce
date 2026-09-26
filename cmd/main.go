package main

import (
	"database/sql"
	"gocommerce/internal/handler"
	"gocommerce/internal/middleware"
	"gocommerce/internal/repository"
	"gocommerce/internal/service"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq" // Driver para PostgreSQL
)

func main() {
	// Carrega variáveis do arquivo .env (se existir)
	if err := godotenv.Load(); err != nil {
		log.Println("Aviso: arquivo .env não encontrado, utilizando variáveis de ambiente do sistema.")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@localhost:5432/gocommerce?sslmode=disable"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "minha_chave_secreta"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Configuração do banco de dados
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Erro ao conectar ao banco de dados: %v", err)
	}
	defer db.Close()

	// Inicialização das camadas
	repo := repository.NewUserRepository(db)
	service := service.NewUserService(repo, jwtSecret)

	// Inicializa tabelas de RBAC, Departamentos e seeds de dados automaticamente
	if err := service.InitSchema(); err != nil {
		log.Printf("Aviso ao inicializar schema RBAC: %v\n", err)
	}

	handler := handler.NewUserHandler(service)

	// Configuração do servidor
	server := gin.Default()

	// Middleware de autenticação JWT
	jwtMiddleware := middleware.AuthMiddleware(jwtSecret)

	// 1. Rotas Públicas
	server.POST("/auth/register", handler.CreateUser) // Cadastro de cliente
	server.POST("/auth/login", handler.Login)         // Login geral

	// 2. Rotas Protegidas
	api := server.Group("/api", jwtMiddleware)
	{
		// Auto-gestão do perfil (Qualquer usuário logado)
		api.GET("/me", middleware.RequirePermission("users:self:read"), handler.GetProfile)
		api.PATCH("/users/:id/password", middleware.RequirePermission("users:self:write"), handler.UpdatePassword)
		api.DELETE("/me", middleware.RequirePermission("users:self:delete"), handler.DeleteOwnAccount)

		// Gestão Departamental (Gerentes gerenciam colaboradores da sua própria área)
		dept := api.Group("/department")
		{
			dept.GET("/users", middleware.RequirePermission("dept_users:read"), handler.ListDepartmentUsers)
			dept.POST("/users", middleware.RequirePermission("dept_users:create"), handler.CreateStaff)
			dept.PATCH("/users/:id", middleware.RequirePermission("dept_users:write"), handler.UpdateUser)
			dept.DELETE("/users/:id", middleware.RequirePermission("dept_users:delete"), handler.DeleteUser)
		}

		// Gestão Global do Sistema (Exclusivo Administrador Geral)
		admin := api.Group("/admin")
		{
			admin.GET("/users", middleware.RequirePermission("users:read_all"), handler.ListUsers)
			admin.GET("/users/:id", middleware.RequirePermission("users:read_all"), handler.GetUser)
			admin.PATCH("/users/:id", middleware.RequirePermission("users:write_all"), handler.UpdateUser)
			admin.DELETE("/users/:id", middleware.RequirePermission("users:delete_all"), handler.DeleteUser)
		}

		// Módulo de Estoque e Produtos
		stock := api.Group("/stock", middleware.RequireDepartment("stock"))
		{
			stock.GET("/products", middleware.RequirePermission("products:read"), func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Listagem de produtos (estoque)"})
			})
			stock.POST("/products", middleware.RequirePermission("products:create"), func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Cadastro de produto realizado"})
			})
		}

		// Módulo Financeiro
		finance := api.Group("/finance", middleware.RequireDepartment("finance"))
		{
			finance.GET("/costs", middleware.RequirePermission("finance:read"), func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Relatório de custos do financeiro"})
			})
			finance.POST("/entries", middleware.RequirePermission("finance:write"), func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Lançamento financeiro registrado"})
			})
		}
	}

	// Inicialização do servidor
	server.Run(":" + port)
}
