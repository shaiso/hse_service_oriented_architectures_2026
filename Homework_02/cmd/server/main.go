package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shaiso/marketplace/generated"
	"github.com/shaiso/marketplace/internal/handler"
	"github.com/shaiso/marketplace/internal/middleware"
	"github.com/shaiso/marketplace/internal/repo"
	"github.com/shaiso/marketplace/internal/service"
)

func main() {
	ctx := context.Background()

	postgresURL := os.Getenv("DATABASE_URL")
	if postgresURL == "" {
		postgresURL = "postgres://postgres:postgres@localhost:55432/marketplace?sslmode=disable"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "super-secret-key-change-in-production"
	}

	pool, err := pgxpool.New(ctx, postgresURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// Repos
	productRepo := repo.NewProductRepo(pool)
	orderRepo := repo.NewOrderRepo(pool)
	promoRepo := repo.NewPromoCodeRepo(pool)
	userOpRepo := repo.NewUserOperationRepo(pool)
	userRepo := repo.NewUserRepo(pool)

	// Services
	productService := service.NewProductService(productRepo)
	orderService := service.NewOrderService(orderRepo, productRepo, promoRepo, userOpRepo, 1)
	promoService := service.NewPromoCodeService(promoRepo)
	authService := service.NewAuthService(userRepo, jwtSecret)

	// Handlers
	productHandler := handler.NewProductHandler(productService)
	orderHandler := handler.NewOrderHandler(orderService)
	promoHandler := handler.NewPromoCodeHandler(promoService)
	authHandler := handler.NewAuthHandler(authService)
	h := handler.NewHandler(productHandler, orderHandler, promoHandler, authHandler)

	strictHandler := generated.NewStrictHandler(h, nil)
	mux := generated.Handler(strictHandler)

	// Middleware chain: Logging → Auth → RoleCheck → Handler
	publicPaths := []string{"/auth/"}
	withRoles := middleware.RoleCheck(mux)
	withAuth := middleware.Auth(authService, publicPaths)(withRoles)
	withLogging := middleware.Logging(withAuth)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", withLogging))
}
