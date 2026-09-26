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
	handler := handler.NewUserHandler(service)

	// Configuração do servidor
	server := gin.Default()

	// Middleware de autenticação JWT
	jwtMiddleware := middleware.AuthMiddleware(jwtSecret)

	// Rotas públicas (não precisam de token)
	server.POST("/auth/register", handler.CreateUser)
	server.POST("/auth/login", handler.Login)

	// Rotas protegidas (exigem que o usuário esteja logado via Bearer Token)
	protected := server.Group("/api", jwtMiddleware)
	{
		protected.GET("/users", handler.ListUsers)
		protected.GET("/users/:id", handler.GetUser)
		protected.PUT("/users/:id", handler.UpdateUser)
		protected.DELETE("/users/:id", handler.DeleteUser)
	}

	// Inicialização do servidor
	server.Run(":" + port)
}
