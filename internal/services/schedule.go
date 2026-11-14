package services

import (
	"fmt"
	"time"

	"zapisBot/internal/config"
	"zapisBot/internal/database"
)

// ScheduleService — сервис управления расписанием.
type ScheduleService struct {
	cfg *config.Config
	db  *database.DB
}

// NewScheduleService создаёт новый сервис расписания.
func NewScheduleService(cfg *config.Config, db *database.DB) *ScheduleService {
	return &ScheduleService{cfg: cfg, db: db}
}

// GetTimeSlots возвращает доступные временные слоты для дня.
func (s *ScheduleService) GetTimeSlots() []int {
	return s.cfg.TimeSlots
}

// WorkHours возвращает начало и конец рабочего дня.
func (s *ScheduleService) WorkHours() (int, int) {
	return s.cfg.WorkHourStart, s.cfg.WorkHourEnd
}

// InitSchedule генерирует слоты расписания на ближайшие 30 дней.
// Вызывается при старте бота и периодически для продления горизонтов планирования.
func (s *ScheduleService) InitSchedule() error {
	now := time.Now()
	y, m, d := now.Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, now.Location())

	createdDays := 0
	for i := 0; i < 30; i++ {
		date := start.AddDate(0, 0, i)
		dateStr := date.Format("2006-01-02")

		// Определяем рабочий день (пн-пт = true, сб-вс = false по умолчанию)
		isWorkday := date.Weekday() != time.Saturday && date.Weekday() != time.Sunday

		// Проверяем, есть ли уже записи на эту дату
		existing, err := s.db.GetScheduleGrid(dateStr)
		if err != nil {
			return fmt.Errorf("проверка расписания на %s: %w", dateStr, err)
		}
		if len(existing) > 0 {
			continue // уже инициализировано
		}

		// Создаём слоты для каждого рабочего часа
		for _, hour := range s.cfg.TimeSlots {
			timeSlot := fmt.Sprintf("%02d:00", hour)
			if err := s.db.SetWorkdaySlot(dateStr, timeSlot, false, isWorkday); err != nil {
				return fmt.Errorf("создание слота %s %s: %w", dateStr, timeSlot, err)
			}
		}
		createdDays++
	}
	if createdDays > 0 {
		fmt.Printf("Расписание инициализировано для %d новых дней (%d слотов/день)\n", createdDays, len(s.cfg.TimeSlots))
	}
	return nil
}