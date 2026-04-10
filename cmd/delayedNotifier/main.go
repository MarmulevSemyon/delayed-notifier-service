package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"delayedNotifier/config"
	"delayedNotifier/internal/cache"
	"delayedNotifier/internal/handler"
	"delayedNotifier/internal/queue"
	"delayedNotifier/internal/repository"
	"delayedNotifier/internal/sender"
	"delayedNotifier/internal/service"

	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/ginext"
)

func main() {

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load error: %v", err)
	}

	// бд
	db, err := dbpg.New(cfg.Postgres.DSN(), nil, &dbpg.Options{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	})
	if err != nil {
		log.Fatalf("db init error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Master.PingContext(ctx); err != nil {
		log.Fatalf("db ping error: %v", err)
	}

	//Очередь
	q, err := queue.New(cfg.RabbitMQ.URL)
	if err != nil {
		log.Fatalf("queue init error: %v", err)
	}
	defer q.Close()

	// Кэш
	c := cache.New(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err := c.Ping(context.Background()); err != nil {
		log.Fatalf("redis ping error: %v", err)
	}

	// хранилище
	repo := repository.New(db)
	// отправитель
	snd := sender.New()

	// консьюмер
	go func() {
		if err := q.StartConsumer(context.Background(), repo, snd, c); err != nil {
			log.Printf("consumer stopped: %v", err)
		}
	}()

	// сервис и хендлеры
	svc := service.New(repo, q, c)
	h := handler.New(svc)

	router := ginext.New("debug")
	router.Use(ginext.Logger(), ginext.Recovery())

	router.GET("/health", func(c *ginext.Context) {
		c.JSON(http.StatusOK, ginext.H{
			"status": "ok",
		})
	})

	router.POST("/notify", h.CreateNotification)
	router.GET("/notify/:id", h.GetNotificationByID)
	router.DELETE("/notify/:id", h.CancelNotificationByID)

	router.GET("/notifications", h.ListNotifications)

	router.GET("/", func(c *ginext.Context) {
		c.File("./static/index.html")
	})
	router.Static("/static", "./static")

	addr := fmt.Sprintf(":%d", cfg.AppPort)
	log.Printf("server started on %s", addr)

	if err := router.Run(addr); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
