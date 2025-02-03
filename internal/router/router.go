package router

import (
	"github.com/gin-gonic/gin"
	"github.com/kontentski/chat/internal/auth"
	"github.com/kontentski/chat/internal/handlers"
	"github.com/kontentski/chat/internal/middleware"
	"github.com/kontentski/chat/internal/services"
	"github.com/swaggo/gin-swagger"
	"github.com/swaggo/files"
	_ "github.com/kontentski/chat/docs"
)

type Router struct {
	engine      *gin.Engine
	userService services.ChatRoomService
}

func NewRouter(userService services.ChatRoomService) *Router {
	return &Router{
		engine:      gin.Default(),
		userService: userService,
	}
}

func (r *Router) SetupRoutes() {
	r.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.engine.Static("/homepage", "./homepage")



	// WebSocket endpoint
	r.engine.GET("/ws", middleware.AuthMiddleware(auth.Store), func(ctx *gin.Context) {
		handlers.HandleWebSocket(ctx.Writer, ctx.Request, r.userService)
	})

	// Authentication routes
	r.registerAuthRoutes()

	//Protected routes
	protected := r.engine.Group("/")
	protected.Use(middleware.AuthMiddleware(auth.Store))
	r.registerProtectedRoutes(protected)

}

func (r *Router) registerAuthRoutes() {
	r.engine.GET("/auth", handlers.AuthHandler)
	r.engine.GET("/auth/callback", handlers.CallbackHandler)
	r.engine.GET("/auth/register/", handlers.RegisterHandler)
	r.engine.POST("/auth/register", handlers.RegisterPostHandler)
	r.engine.POST("/auth/logout", handlers.LogoutHandler)
}

func (r *Router) registerProtectedRoutes(rg *gin.RouterGroup) {
	//user routes
	rg.POST("/users", handlers.CreateUser(r.userService))

	// Message routes
	rg.GET("/messages/:chatRoomID", handlers.GetMessagesHandler(r.userService))
	rg.DELETE("/messages/:messageID", handlers.DeleteMessageHandler(r.userService))

	// Chat room routes
	rg.GET("/api/chatrooms", handlers.GetUserChatRoomsHandler(r.userService))
	rg.POST("/api/chatrooms/leave/:chatRoomID", handlers.LeaveTheChatRoomHandler(r.userService))
	rg.GET("/api/chatrooms/search-users", handlers.SearchUsersHandler(r.userService))
	rg.POST("/api/chatrooms/add-user", handlers.AddUserHandler(r.userService))
	rg.POST("/api/upload-media", handlers.UploadMediaHandler(r.userService))
	r.engine.GET("/hello", func(c *gin.Context) {
		c.String(200, "Hello, World!")
	})
}

func (r *Router) Start(addr string) error {
	return r.engine.Run(addr)
}
