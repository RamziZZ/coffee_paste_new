package routes

import (
	"net/http"

	"coffee_paste_backend/config"
	"coffee_paste_backend/internal/auth/handler"
	passwordresetrepository "coffee_paste_backend/internal/auth/passwordreset/repository"
	"coffee_paste_backend/internal/auth/repository"
	"coffee_paste_backend/internal/auth/service"

	favoritehandler "coffee_paste_backend/internal/favorites/handler"
	favoriterepository "coffee_paste_backend/internal/favorites/repository"
	favoriteservice "coffee_paste_backend/internal/favorites/service"

	inventoryhandler "coffee_paste_backend/internal/inventory/handler"
	inventoryrepository "coffee_paste_backend/internal/inventory/repository"
	inventoryservice "coffee_paste_backend/internal/inventory/service"

	menuhandler "coffee_paste_backend/internal/menu/handler"
	menurepository "coffee_paste_backend/internal/menu/repository"
	menuservice "coffee_paste_backend/internal/menu/service"

	orderitemhandler "coffee_paste_backend/internal/order_items/handler"
	orderitemrepository "coffee_paste_backend/internal/order_items/repository"
	orderitemservice "coffee_paste_backend/internal/order_items/service"

	orderhandler "coffee_paste_backend/internal/orders/handler"
	orderrepository "coffee_paste_backend/internal/orders/repository"
	orderservice "coffee_paste_backend/internal/orders/service"

	paymenthandler "coffee_paste_backend/internal/payments/handler"
	paymentrepository "coffee_paste_backend/internal/payments/repository"
	paymentservice "coffee_paste_backend/internal/payments/service"

	midtranshandler "coffee_paste_backend/internal/payments/midtrans/handler"
	midtransrepository "coffee_paste_backend/internal/payments/midtrans/repository"
	midtransservice "coffee_paste_backend/internal/payments/midtrans/service"

	notificationhandler "coffee_paste_backend/internal/notifications/handler"
	notificationrepository "coffee_paste_backend/internal/notifications/repository"
	notificationservice "coffee_paste_backend/internal/notifications/service"

	reporthandler "coffee_paste_backend/internal/reports/handler"
	reportrepository "coffee_paste_backend/internal/reports/repository"
	reportservice "coffee_paste_backend/internal/reports/service"

	reviewhandler "coffee_paste_backend/internal/reviews/handler"
	reviewrepository "coffee_paste_backend/internal/reviews/repository"
	reviewservice "coffee_paste_backend/internal/reviews/service"

	userhandler "coffee_paste_backend/internal/users/handler"
	userrepository "coffee_paste_backend/internal/users/repository"
	userservice "coffee_paste_backend/internal/users/service"

	shifthandler "coffee_paste_backend/internal/shifts/handler"
	shiftrepository "coffee_paste_backend/internal/shifts/repository"
	shiftservice "coffee_paste_backend/internal/shifts/service"

	"coffee_paste_backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupRoutes(
	router *gin.Engine,
	db *pgxpool.Pool,
	cfg *config.Config,
) {
	// =========================
	// AUTH DEPENDENCIES
	// =========================
	authRepository := repository.NewAuthRepository(db)
	passwordResetRepository := passwordresetrepository.NewPasswordResetRepository(db)
	authService := service.NewAuthService(authRepository, passwordResetRepository, *cfg)
	authHandler := handler.NewAuthHandler(authService)

	// =========================
	// MENU DEPENDENCIES
	// =========================
	menuRepository := menurepository.NewMenuRepository(db)
	menuService := menuservice.NewMenuService(menuRepository)
	menuHandler := menuhandler.NewMenuHandler(menuService)

	// =========================
	// NOTIFICATIONS DEPENDENCIES
	// =========================
	notificationRepository := notificationrepository.NewNotificationRepository(db)
	notificationService := notificationservice.NewNotificationService(
		notificationRepository,
	)
	notificationHandler := notificationhandler.NewNotificationHandler(
		notificationService,
	)

	// =========================
	// FAVORITES DEPENDENCIES
	// =========================
	favoriteRepository := favoriterepository.NewFavoriteRepository(db)
	favoriteService := favoriteservice.NewFavoriteService(favoriteRepository)
	favoriteHandler := favoritehandler.NewFavoriteHandler(favoriteService)

	// =========================
	// ORDERS DEPENDENCIES
	// =========================
	orderRepository := orderrepository.NewOrderRepository(db)

	// =========================
	// INVENTORY DEPENDENCIES
	// =========================
	inventoryRepository := inventoryrepository.NewInventoryRepository(db)
	inventoryService := inventoryservice.NewInventoryService(
		inventoryRepository,
		notificationService,
	)
	inventoryHandler := inventoryhandler.NewInventoryHandler(inventoryService)

	// Orders memakai inventory sebagai port stok (cek saat membuat,
	// potong saat pesanan selesai) dan notifications untuk update status.
	orderService := orderservice.NewOrderService(
		orderRepository,
		inventoryService,
		notificationService,
	)
	orderHandler := orderhandler.NewOrderHandler(orderService)

	// =========================
	// ORDER ITEMS DEPENDENCIES
	// =========================
	orderItemRepository := orderitemrepository.NewOrderItemRepository(db)
	orderItemService := orderitemservice.NewOrderItemService(orderItemRepository)
	orderItemHandler := orderitemhandler.NewOrderItemHandler(orderItemService)

	// =========================
	// PAYMENTS DEPENDENCIES
	// =========================
	paymentRepository := paymentrepository.NewPaymentRepository(db)
	paymentService := paymentservice.NewPaymentService(paymentRepository)
	paymentHandler := paymenthandler.NewPaymentHandler(paymentService)

	// =========================
	// MIDTRANS DEPENDENCIES
	// =========================
	midtransRepository := midtransrepository.NewMidtransRepository(db)
	midtransService := midtransservice.NewMidtransService(
		midtransRepository,
		*cfg,
		notificationService,
	)
	midtransHandler := midtranshandler.NewMidtransHandler(midtransService)

	// =========================
	// REVIEWS DEPENDENCIES
	// =========================
	reviewRepository := reviewrepository.NewReviewRepository(db)
	reviewService := reviewservice.NewReviewService(reviewRepository)
	reviewHandler := reviewhandler.NewReviewHandler(reviewService)

	// =========================
	// REPORTS DEPENDENCIES
	// =========================
	reportRepository := reportrepository.NewReportRepository(db)
	reportService := reportservice.NewReportService(reportRepository)
	reportHandler := reporthandler.NewReportHandler(reportService)

	// =========================
	// USERS DEPENDENCIES
	// =========================
	userRepository := userrepository.NewUserRepository(db)
	userService := userservice.NewUserService(userRepository)
	userHandler := userhandler.NewUserHandler(userService)

	// =========================
	// SHIFTS DEPENDENCIES
	// =========================
	shiftRepository := shiftrepository.NewShiftRepository(db)
	shiftService := shiftservice.NewShiftService(shiftRepository)
	shiftHandler := shifthandler.NewShiftHandler(shiftService)

	// =========================
	// API V1
	// =========================
	api := router.Group("/api/v1")

	// =========================
	// HEALTH CHECK
	// =========================
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Coffee Paste Backend is running",
		})
	})

	// =========================
	// MENU PUBLIC & ADMIN
	// =========================
	menu := api.Group("/menu")
	{
		// Public daftar dan detail menu
		menu.GET("", menuHandler.GetAllMenu)
		menu.GET("/:id", menuHandler.GetMenuByID)
		menu.GET("/featured", menuHandler.GetFeaturedMenu)
		menu.GET("/categories", menuHandler.GetCategories)
		menu.GET("/filters", menuHandler.GetFilters)

		// ADMIN CRUD menu
		menu.POST(
			"",
			middleware.AuthMiddleware(cfg),
			middleware.RequireRole("admin"),
			menuHandler.CreateMenu,
		)
		menu.PUT(
			"/:id",
			middleware.AuthMiddleware(cfg),
			middleware.RequireRole("admin"),
			menuHandler.UpdateMenu,
		)
		menu.DELETE(
			"/:id",
			middleware.AuthMiddleware(cfg),
			middleware.RequireRole("admin"),
			menuHandler.DeleteMenu,
		)
	}

	// =========================
	// AUTH PUBLIC
	// =========================
	auth := api.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/forgot-password", authHandler.ForgotPassword)
		auth.POST("/reset-password", authHandler.ResetPassword)
		auth.POST("/register", authHandler.RegisterCustomer)
	}

	// =========================
	// AUTH PROTECTED
	// =========================
	protectedAuth := api.Group("/auth")
	protectedAuth.Use(middleware.AuthMiddleware(cfg))
	{
		protectedAuth.POST("/logout", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "Logout berhasil",
			})
		})

		protectedAuth.GET("/me", authHandler.Me)
	}

	// =========================
	// FAVORITES PROTECTED
	// =========================
	favorites := api.Group("/favorites")
	favorites.Use(middleware.AuthMiddleware(cfg))
	{
		favorites.GET("/", favoriteHandler.GetFavorites)
		favorites.POST("/:menu_id", favoriteHandler.CreateFavorite)
		favorites.DELETE("/:menu_id", favoriteHandler.DeleteFavorite)
	}

	// =========================
	// ORDERS PROTECTED
	// =========================
	orders := api.Group("/orders")
	orders.Use(middleware.AuthMiddleware(cfg))
	{
		// Order CRUD dan status
		orders.POST("", orderHandler.CreateOrder)
		orders.GET("", orderHandler.ListOrders)
		orders.GET("/:id", orderHandler.GetOrder)
		orders.POST("/:id/cancel", orderHandler.CancelOrder)

		orders.PATCH(
			"/:id/status",
			middleware.RequireRole("kasir", "admin"),
			orderHandler.UpdateStatus,
		)

		// Payments berdasarkan order
		orders.GET("/:id/payments", paymentHandler.GetByOrderID)

		// Order Items
		orders.GET(
			"/:id/items",
			func(c *gin.Context) {
				c.Params = append(c.Params, gin.Param{
					Key:   "order_id",
					Value: c.Param("id"),
				})
				orderItemHandler.GetByOrderID(c)
			},
		)

		orders.POST(
			"/:id/items",
			middleware.RequireRole("kasir", "admin"),
			func(c *gin.Context) {
				c.Params = append(c.Params, gin.Param{
					Key:   "order_id",
					Value: c.Param("id"),
				})
				orderItemHandler.Create(c)
			},
		)

		orders.PUT(
			"/:id/items/:item_id",
			middleware.RequireRole("kasir", "admin"),
			func(c *gin.Context) {
				c.Params = append(c.Params, gin.Param{
					Key:   "order_id",
					Value: c.Param("id"),
				})
				orderItemHandler.Update(c)
			},
		)

		orders.DELETE(
			"/:id/items/:item_id",
			middleware.RequireRole("kasir", "admin"),
			func(c *gin.Context) {
				c.Params = append(c.Params, gin.Param{
					Key:   "order_id",
					Value: c.Param("id"),
				})
				orderItemHandler.Delete(c)
			},
		)
	}

	// =========================
	// PAYMENTS PROTECTED
	// =========================
	payments := api.Group("/payments")
	payments.Use(middleware.AuthMiddleware(cfg))
	{
		payments.POST(
			"",
			middleware.RequireRole("kasir", "admin"),
			paymentHandler.Create,
		)

		payments.GET("/:id", paymentHandler.GetByID)

		payments.PATCH(
			"/:id/status",
			middleware.RequireRole("kasir", "admin"),
			paymentHandler.UpdateStatus,
		)
	}

	// =========================
	// MIDTRANS PAYMENT GATEWAY
	// =========================

	// Config Midtrans untuk frontend (client key & environment)
	api.GET("/payments/midtrans/config", midtransHandler.GetConfig)

	// Webhook notifikasi dari Midtrans (tanpa auth, diverifikasi via signature)
	api.POST("/payments/midtrans/webhook", midtransHandler.HandleNotification)

	// Endpoint Midtrans yang butuh login
	midtransRoutes := api.Group("/payments/midtrans")
	midtransRoutes.Use(middleware.AuthMiddleware(cfg))
	{
		// Buat snap transaction untuk sebuah pesanan
		midtransRoutes.POST(
			"/snap",
			middleware.RequireRole("customer", "kasir", "admin"),
			midtransHandler.CreateSnap,
		)

		// Cek status pembayaran Midtrans berdasarkan order_id
		midtransRoutes.GET(
			"/order/:order_id",
			midtransHandler.GetByOrderID,
		)

		// Detail pembayaran berdasarkan ID
		midtransRoutes.GET(
			"/:id",
			midtransHandler.GetByID,
		)
	}

	// =========================
	// USERS PROTECTED
	// =========================
	users := api.Group("/users")
	users.Use(middleware.AuthMiddleware(cfg))
	{
		// Semua user yang sudah login
		users.GET("/me", userHandler.Me)
		users.PUT("/me", userHandler.UpdateMe)

		// Khusus admin
		adminUsers := users.Group("")
		adminUsers.Use(middleware.RequireRole("admin"))

		adminUsers.GET("", userHandler.List)
		adminUsers.GET("/:id", userHandler.GetByID)
		adminUsers.PATCH("/:id/status", userHandler.UpdateStatus)
	}

	// =========================
	// SHIFTS PROTECTED
	// =========================
	shifts := api.Group("/shifts")
	shifts.Use(middleware.AuthMiddleware(cfg))
	{
		// Membuka shift kasir
		shifts.POST("", shiftHandler.Open)

		// Melihat shift aktif milik sendiri
		shifts.GET("/active", shiftHandler.GetActive)

		// Melihat riwayat shift milik sendiri
		shifts.GET("", shiftHandler.ListMine)

		// Menutup shift milik sendiri
		shifts.PATCH("/:id/close", shiftHandler.Close)
	}

	// =========================
	// INVENTORY PROTECTED (kasir & admin)
	// =========================
	inventory := api.Group("/inventory")
	inventory.Use(middleware.AuthMiddleware(cfg))
	{
		staff := middleware.RequireRole("kasir", "admin")

		inventory.GET("", staff, inventoryHandler.List)
		inventory.GET("/:menu_id", staff, inventoryHandler.Get)
		inventory.POST("/adjust", staff, inventoryHandler.Adjust)
	}

	// =========================
	// REVIEWS PROTECTED
	// =========================
	reviews := api.Group("/reviews")
	reviews.Use(middleware.AuthMiddleware(cfg))
	{
		// Customer memberi rating untuk pesanannya yang selesai
		reviews.POST(
			"",
			middleware.RequireRole("customer"),
			reviewHandler.Create,
		)

		// Customer melihat review miliknya sendiri
		reviews.GET(
			"/mine",
			middleware.RequireRole("customer"),
			reviewHandler.ListMine,
		)

		// Staf melihat semua review dengan filter
		reviews.GET(
			"",
			middleware.RequireRole("kasir", "admin"),
			reviewHandler.ListAll,
		)

		reviews.GET("/:id", reviewHandler.Get)

		// Moderasi review
		reviews.DELETE(
			"/:id",
			middleware.RequireRole("admin"),
			reviewHandler.Delete,
		)
	}

	// =========================
	// NOTIFICATIONS PROTECTED
	// =========================
	notifications := api.Group("/notifications")
	notifications.Use(middleware.AuthMiddleware(cfg))
	{
		notifications.GET("", notificationHandler.List)
		notifications.GET("/unread-count", notificationHandler.UnreadCount)
		notifications.PATCH("/:id/read", notificationHandler.MarkRead)
		notifications.PATCH("/read-all", notificationHandler.MarkAllRead)
	}

	// =========================
	// REPORTS PROTECTED (admin)
	// =========================
	reports := api.Group("/reports")
	reports.Use(middleware.AuthMiddleware(cfg))
	{
		adminOnly := middleware.RequireRole("admin")

		// Laporan penjualan (JSON)
		reports.GET("/sales", adminOnly, reportHandler.GetSalesReport)

		// Ekspor PDF laporan penjualan
		reports.GET("/sales/pdf", adminOnly, reportHandler.ExportSalesPDF)
	}

}
