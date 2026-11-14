package bot

import (
	"sync"
	"time"
)

// FSMState — состояние конечного автомата клиента.
type FSMState string

const (
	StateStart               FSMState = "start"                  // приветствие / главное меню
	StateRegistration        FSMState = "registration"           // ввод имени
	StateMainMenu            FSMState = "main_menu"              // главное меню
	StateOrderCategory       FSMState = "order_category"         // выбор категории услуги
	StateOrderItemType       FSMState = "order_item_type"        // выбор типа изделия
	StateOrderParameters     FSMState = "order_parameters"       // ввод параметров (радиус / габариты)
	StateOrderColor          FSMState = "order_color"            // выбор цвета RAL
	StateOrderCoating        FSMState = "order_coating"          // выбор покрытия (матовый/глянец)
	StateOrderDate           FSMState = "order_date"             // выбор даты
	StateOrderTime           FSMState = "order_time"             // выбор времени
	StateOrderCalculator     FSMState = "order_calculator"       // показ калькулятора
	StateOrderConfirm        FSMState = "order_confirm"          // подтверждение заказа
	StateOrderContact        FSMState = "order_contact"          // запрос контакта (телефон)
	StateRescheduleDate      FSMState = "reschedule_date"        // выбор новой даты при переносе
	StateRescheduleTime      FSMState = "reschedule_time"        // выбор нового времени при переносе
	StateAdminMenu           FSMState = "admin_menu"             // админ-меню
	StateAdminPriceCRUD      FSMState = "admin_price_crud"       // CRUD прайс-листа
	StateAdminWorkday        FSMState = "admin_workday"          // настройка рабочих дней
)

// UserSession — сессия пользователя в процессе заполнения заказа.
type UserSession struct {
	mu        sync.Mutex
	ChatID    int64
	State     FSMState
	Data      map[string]string
	UpdatedAt time.Time
	CreatedAt time.Time
	ClientID  int64  // ID клиента в БД (после регистрации)
	Phone     string // телефон клиента
	FullName  string // ФИО клиента
}

// NewUserSession создаёт новую сессию.
func NewUserSession(chatID int64) *UserSession {
	now := time.Now()
	return &UserSession{
		ChatID:    chatID,
		State:     StateStart,
		Data:      make(map[string]string),
		UpdatedAt: now,
		CreatedAt: now,
	}
}

// Lock блокирует сессию для безопасной работы из нескольких горутин.
func (s *UserSession) Lock() {
	s.mu.Lock()
}

// Unlock снимает блокировку сессии.
func (s *UserSession) Unlock() {
	s.mu.Unlock()
}

// SetData безопасно сохраняет значение в Data.
func (s *UserSession) SetData(key, val string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Data == nil {
		s.Data = make(map[string]string)
	}
	s.Data[key] = val
	s.UpdatedAt = time.Now()
}

// GetData безопасно получает значение из Data.
func (s *UserSession) GetData(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Data[key]
}

// ResetOrderData очищает временные данные заказа.
func (s *UserSession) ResetOrderData() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data = make(map[string]string)
	s.UpdatedAt = time.Now()
}

// IsOrderState возвращает true, если сессия находится в процессе оформления заказа.
func (s *UserSession) IsOrderState() bool {
	switch s.State {
	case StateOrderCategory, StateOrderItemType, StateOrderParameters,
		StateOrderColor, StateOrderCoating, StateOrderDate, StateOrderTime,
		StateOrderCalculator, StateOrderConfirm, StateOrderContact:
		return true
	default:
		return false
	}
}
