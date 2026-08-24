package routes

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"auth-session/internal/middleware"
	user "auth-session/internal/users"
)

func NewUserRouter(
	userHandler *user.UserHandler,
	authMiddleware  *middleware.AuthMiddleware,
) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", userHandler.RegisterUser)
			auth.POST("/login",    userHandler.LoginUser)
			auth.POST("/refresh",  userHandler.RefreshToken)
			auth.POST("/logout",   authMiddleware.Authenticate(), userHandler.LogoutUser)
		}

		users := v1.Group("/users/me", authMiddleware.Authenticate())
		{
			users.GET("",   userHandler.GetMyProfile)
			users.PATCH("", userHandler.UpdateMyProfile)
		}

		sessions := v1.Group("users/me/sessions", authMiddleware.Authenticate())
		{
			sessions.GET("",           userHandler.GetActiveSessionsMyProfile)
			sessions.DELETE("/:sid",   userHandler.RevokeSessionMyProfile)
			sessions.DELETE("/others", userHandler.RevokeAllOtherSessionsMyProfile)
			sessions.DELETE("",        userHandler.RevokeAllSessionsMyProfile)
		}

		admin := v1.Group("/admin", authMiddleware.Authenticate(), authMiddleware.AdminSecretMiddleware())
		{
			admin.GET("/users",                      userHandler.GetUsers)
			admin.GET("/users/:sub",                 userHandler.GetUserByID)
			admin.PATCH("/users/:sub",               userHandler.UpdateUser)
			admin.GET("/users/:sub/sessions",        userHandler.GetActiveSessionsUser)
			admin.DELETE("users/:sub/sessions/:sid", userHandler.RevokeSessionUser)
			admin.DELETE("users/:sub/sessions",      userHandler.RevokeAllSessionsUser)
		}
	}

	return r
}