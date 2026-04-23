package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/akito-0520/mago-ai/backend/internal/config"
	"github.com/akito-0520/mago-ai/backend/internal/interface/http/handler"
	appmw "github.com/akito-0520/mago-ai/backend/internal/interface/http/middleware"
)

func main() {
	// 環境変数の読み込み
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	// Echoインスタンスの作成
	e := echo.New()

	// middleware の登録
	e.Use(middleware.RequestLogger()) // リクエストをログで出力
	e.Use(middleware.Recover())       // panicが起きた時に500エラーに変換
	e.Use(appmw.Timing())             // リクエストの処理時間をログで出力

	// ルートの登録
	e.GET("/healthz", handler.Health)

	// サーバー起動
	go func() {
		if err := e.Start(":" + cfg.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	// ctrlまたはkillをリッスンする
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	// contextを作成してGraceful Shutdown を行う
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	log.Print("start shutdown")
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Println("shutdown error: ", err)
	}
}
