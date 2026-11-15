package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"

	"zapisBot/internal/bot"
	"zapisBot/internal/bot/handlers"
	"zapisBot/internal/config"
	"zapisBot/internal/database"
	"zapisBot/internal/services"
)

func main() {
	// Загрузка переменных окружения
	if err := godotenv.Load(); err != nil {
		if err2 := godotenv.Load("cmd/bot/.env"); err2 != nil {
			log.Println("Предупреждение: .env файл не найден, используются переменные окружения системы")
		}
	}

	// Загрузка конфигурации
	cfg := config.Load()
	log.Printf("Конфигурация загружена. Бот: %s, Админов: %d", cfg.BotName, len(cfg.AdminIDs))

	// Подключение к базе данных
	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Не удалось подключиться к БД: %v", err)
	}
	defer db.Close()

	// Инициализация бота Telegram
	tgBot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		log.Fatalf("Не удалось инициализировать бота: %v", err)
	}
	tgBot.Debug = false
	log.Printf("Авторизован как %s", tgBot.Self.UserName)

	// Инициализация сервисов
	sessionStore := bot.NewSessionStore()
	calc := services.NewCalculator(cfg, db)
	orderSvc := services.NewOrderService(cfg, db)
	scheduleSvc := services.NewScheduleService(cfg, db)
	notifySvc := services.NewNotificationService(cfg, db, tgBot)

	// Инициализация расписания (создание слотов на 30 дней вперёд)
	if err := scheduleSvc.InitSchedule(); err != nil {
		log.Fatalf("Не удалось инициализировать расписание: %v", err)
	}

	// Инициализация хендлеров
	clientHandler := handlers.NewClientHandler(cfg, db, tgBot, sessionStore, calc, orderSvc, scheduleSvc, notifySvc)
	adminHandler := handlers.NewAdminHandler(cfg, db, tgBot, sessionStore)

	// Контекст с graceful shutdown по SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Канал обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := tgBot.GetUpdatesChan(u)

	// Фоновая задача: очистка старых сессий каждые 30 минут
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				removed := sessionStore.Cleanup(24 * time.Hour)
				if removed > 0 {
					log.Printf("Очищено сессий: %d", removed)
				}
			}
		}
	}()

	// Фоновая задача: продление расписания на 30 дней вперёд раз в 6 часов
	go func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := scheduleSvc.InitSchedule(); err != nil {
					log.Printf("Ошибка продления расписания: %v", err)
				}
			}
		}
	}()

	// Фоновая задача: проверка напоминаний каждые 5 минут
	if cfg.NotificationEnabled {
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := notifySvc.CheckReminders(); err != nil {
						log.Printf("Ошибка проверки напоминаний: %v", err)
					}
				}
			}
		}()
	}

	log.Println("Бот запущен. Нажмите Ctrl+C для остановки.")

	// Основной цикл обработки обновлений
	for {
		select {
		case <-ctx.Done():
			log.Println("Получен сигнал остановки, завершение работы...")
			return
		case update, ok := <-updates:
			if !ok {
				log.Println("Канал обновлений закрыт")
				return
			}
			if update.Message != nil {
				msg := update.Message
				if msg.From == nil {
					continue
				}
				userID := msg.From.ID
				chatID := msg.Chat.ID

				// Если админ и в админ-состоянии или команда /admin — обрабатываем через админ-хендлер
				if cfg.IsAdmin(userID) {
					session := sessionStore.Get(chatID)
					session.Lock()
					state := session.State
					session.Unlock()

					if msg.Text == "/admin" ||
						state == bot.StateAdminPriceCRUD ||
						state == bot.StateAdminWorkday ||
						state == bot.StateAdminMenu {
						go adminHandler.HandleMessage(update)
						continue
					}
				}

				go clientHandler.HandleMessage(update)
			}
			if update.CallbackQuery != nil {
				cb := update.CallbackQuery
				if cb.From != nil && cb.Message != nil && cfg.IsAdmin(cb.From.ID) && len(cb.Data) >= 4 && cb.Data[:4] == "adm:" {
					go adminHandler.HandleCallback(update)
					continue
				}
				go clientHandler.HandleCallback(update)
			}
		}
	}
}
