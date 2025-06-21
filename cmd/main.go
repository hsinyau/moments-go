package main

import (
	"log"
	"time"

	"moments-go/config"
	"moments-go/handlers"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	// 加载配置
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建机器人实例
	bot, err := tgbotapi.NewBotAPI(config.Cfg.TelegramBotToken)
	if err != nil {
		log.Fatalf("创建机器人失败: %v", err)
	}

	bot.Debug = false
	log.Printf("机器人已启动: %s", bot.Self.UserName)

	// 设置更新配置
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	// 持续运行，处理网络错误
	for {
		log.Println("正在连接 Telegram API...")
		
		// 获取更新通道
		updates := bot.GetUpdatesChan(updateConfig)

		// 处理更新
		for update := range updates {
			log.Printf("收到更新: %+v", update)

			var err error

			// 处理回调查询（标签选择）
			if update.CallbackQuery != nil {
				err = handlers.HandleCallbackQuery(bot, update)
				if err != nil {
					log.Printf("处理回调查询失败: %v", err)
				}
				continue
			}

			// 处理普通消息
			if update.Message == nil {
				continue
			}

			log.Printf("收到消息: [%d] %s", update.Message.Chat.ID, update.Message.Text)

			// 根据消息类型处理
			switch {
			case update.Message.Photo != nil:
				err = handlers.HandlePhotoMessage(bot, update)
			case update.Message.Video != nil:
				err = handlers.HandleVideoMessage(bot, update)
			case update.Message.Text != "":
				err = handlers.HandleTextMessage(bot, update)
			}

			if err != nil {
				log.Printf("处理消息失败: %v", err)
			}
		}

		// 如果更新通道关闭，说明连接断开，等待后重试
		log.Println("连接断开，3秒后重试...")
		time.Sleep(3 * time.Second)
	}
} 
