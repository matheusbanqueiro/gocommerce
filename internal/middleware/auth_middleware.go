package middleware

import (
	"net/http"
	"strings"
	"sync"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

// tokenBlacklist armazena tokens JWT invalidados em memória de forma thread-safe.
var tokenBlacklist sync.Map

// InvalidateToken adiciona o token JWT à lista de revogação.
func InvalidateToken(tokenString string) {
	tokenBlacklist.Store(tokenString, true)
}

// IsTokenInvalidated verifica se o token está na lista de revogados.
func IsTokenInvalidated(tokenString string) bool {
	_, found := tokenBlacklist.Load(tokenString)
	return found
}

// AuthMiddleware é o middleware base que valida o JWT e injeta claims (userID, role, department, level, permissions).
func AuthMiddleware(secretKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "cabeçalho de autorização não informado"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "formato de token inválido (esperado: Bearer <token>)"})
			c.Abort()
			return
		}

		// Verifica se o token já foi invalidado/revogado
		if IsTokenInvalidated(tokenString) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token foi revogado e não é mais válido"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.NewValidationError("unexpected signing method", jwt.ValidationErrorSignatureInvalid)
			}
			return []byte(secretKey), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido ou expirado"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("userID", claims["userID"])
			c.Set("role", claims["role"])

			if dept, exists := claims["department"]; exists {
				if deptStr, ok := dept.(string); ok {
					c.Set("department", deptStr)
				}
			}

			if lvl, exists := claims["level"]; exists {
				if lvlFloat, ok := lvl.(float64); ok {
					c.Set("level", int(lvlFloat))
				}
			}

			var permissions []string
			if perms, exists := claims["permissions"]; exists {
				if permList, ok := perms.([]interface{}); ok {
					for _, p := range permList {
						if pStr, ok := p.(string); ok {
							permissions = append(permissions, pStr)
						}
					}
				}
			}
			c.Set("permissions", permissions)
			c.Set("rawToken", tokenString)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "claims do token inválidas"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermission garante que o usuário possui uma permissão atômica específica.
func RequirePermission(requiredPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		permsVal, exists := c.Get("permissions")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "acesso negado: permissões não encontradas"})
			c.Abort()
			return
		}

		permissions, ok := permsVal.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "acesso negado: formato de permissões inválido"})
			c.Abort()
			return
		}

		for _, p := range permissions {
			if p == requiredPermission {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "acesso negado: requer a permissão '" + requiredPermission + "'"})
		c.Abort()
	}
}

// RequireDepartment garante que o usuário pertence a uma área específica ou seja admin geral.
func RequireDepartment(allowedDepartments ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userDept := c.GetString("department")
		userLevel := 1
		if lvl, exists := c.Get("level"); exists {
			if l, ok := lvl.(int); ok {
				userLevel = l
			}
		}

		// Admin Geral (nível 100) tem acesso universal a todos os departamentos
		if userLevel >= 100 {
			c.Next()
			return
		}

		for _, d := range allowedDepartments {
			if userDept == d {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "acesso negado: este recurso pertence a outro departamento"})
		c.Abort()
	}
}
