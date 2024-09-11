package server

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"goflare.io/payment/handlers"
)

type Server struct {
	echo          *echo.Echo
	Customer      handlers.CustomerHandler
	Product       handlers.ProductHandler
	Price         handlers.PriceHandler
	PaymentIntent handlers.PaymentIntentHandler
	Webhook       handlers.WebhookHandler
}

func NewServer(
	Customer handlers.CustomerHandler,
	Product handlers.ProductHandler,
	Price handlers.PriceHandler,
	PaymentIntent handlers.PaymentIntentHandler,
	Webhook handlers.WebhookHandler,
) *Server {
	return &Server{
		echo:          echo.New(),
		Customer:      Customer,
		Product:       Product,
		Price:         Price,
		Webhook:       Webhook,
		PaymentIntent: PaymentIntent,
	}
}

// Start initializes the server by registering middlewares and routes, and starts listening for connections on the provided address.
// It returns an error if there is an issue starting the server.
func (s *Server) Start(address string) error {
	s.registerMiddlewares()
	s.registerRoutes()
	return s.echo.Start(address)
}

// Run starts the server by calling the Start method in a goroutine. If an error occurs, it
// logs the error and terminates the server. It then listens for an OS interrupt signal or a SIGTERM
// signal to gracefully shut down the server. Once the signal is received, it creates a context with
// a timeout of 5 seconds, cancels the context after the method returns, and returns the result of
// shutting down the server.
func (s *Server) Run(address string) error {

	go func() {
		if err := s.Start(address); err != nil {
			s.echo.Logger.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.echo.Shutdown(ctx)
}

func (s *Server) registerMiddlewares() {
	s.echo.Use(middleware.Recover())
}

func (s *Server) registerRoutes() {

	s.echo.POST("/customer", s.Customer.CreateCustomer)
	s.echo.GET("/customer/:id", s.Customer.GetCustomer)
	s.echo.PUT("/customer/:id", s.Customer.UpdateCustomer)
	s.echo.DELETE("/customer/:id", s.Customer.DeleteCustomer)

	s.echo.POST("/product", s.Product.CreateProduct)
	s.echo.GET("/product/:id", s.Product.GetProduct)
	s.echo.PUT("/product/:id", s.Product.UpdateProduct)
	s.echo.DELETE("/product/:id", s.Product.DeleteProduct)
	s.echo.GET("/product", s.Product.ListProducts)

	s.echo.POST("/price", s.Price.CreatePrice)
	s.echo.DELETE("/price/:id", s.Price.DeletePrice)

	s.echo.POST("/payment/intent", s.PaymentIntent.CreatePaymentIntent)
	s.echo.POST("/payment/intent/confirm", s.PaymentIntent.ConfirmPaymentIntent)

	s.echo.POST("/webhook", s.Webhook.HandleWebhook)
}
