package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// Структура задачи
type Task struct {
	ID        int       `json:"id"`         // Идентификатор
	Title     string    `json:"title"`      // Название задачи
	Completed bool      `json:"completed"`  // Статус выполнения
	CreatedAt time.Time `json:"created_at"` // Время создания
}

type TaskStorage struct {
	mu     sync.RWMutex
	tasks  map[int]Task
	nextID int
}

func NewTaskStorage() *TaskStorage {
	return &TaskStorage{
		tasks:  make(map[int]Task),
		nextID: 1,
	}
}

func (s *TaskStorage) Create(task Task) Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	task.ID = s.nextID
	task.CreatedAt = time.Now()
	s.tasks[task.ID] = task
	s.nextID++
	return task
}

func (s *TaskStorage) GetByID(id int) (Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, exists := s.tasks[id]
	return task, exists
}

func (s *TaskStorage) GetAll(completed *bool) []Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := []Task{}
	for _, task := range s.tasks {
		if completed == nil || task.Completed == *completed {
			result = append(result, task)
		}
	}
	return result
}

type TaskService struct {
	store   *TaskStorage  // Ссылка на хранилище
	logChan chan<- string // Канал для логов
}

// Конструктор сервиса
func NewTaskService(store *TaskStorage, logChan chan<- string) *TaskService {
	return &TaskService{store, logChan}
}

type TaskHandler struct {
	service *TaskService
}

// Конструктор обработчика
func NewTaskHandler(service *TaskService) *TaskHandler {
	return &TaskHandler{service}
}

// Обработчик GET /tasks
func (h *TaskHandler) GetTasks(w http.ResponseWriter, r *http.Request) {
	// Парсинг параметра фильтра
	var filter *bool
	if param := r.URL.Query().Get("completed"); param != "" {
		val, err := strconv.ParseBool(param)
		if err == nil {
			filter = &val
		}
	}
	// Получение задач из хранилища
	tasks := h.service.store.GetAll(filter)
	// Асинхронное логирование
	h.service.logChan <- "Запрос всех задач: найдено " + strconv.Itoa(len(tasks))
	// Формирование ответа
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// Обработчик GET /tasks/{id}
func (h *TaskHandler) GetTaskByID(w http.ResponseWriter, r *http.Request) {
	// Парсинг ID из URL
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Некорректный ID", http.StatusBadRequest)
		return
	}
	// Поиск задачи
	task, exists := h.service.store.GetByID(id)
	if !exists {
		http.Error(w, "Задача не найдена", http.StatusNotFound)
		return
	}
	// Асинхронное логирование
	h.service.logChan <- "Запрос задачи #" + strconv.Itoa(id)
	// Формирование ответа
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// Обработчик POST /tasks
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	// Декодирование тела запроса
	var newTask Task
	if err := json.NewDecoder(r.Body).Decode(&newTask); err != nil {
		http.Error(w, "Неверный формат данных", http.StatusBadRequest)
		return
	}
	// Валидация
	if newTask.Title == "" {
		http.Error(w, "Название задачи обязательно", http.StatusBadRequest)
		return
	}
	// Создание задачи
	createdTask := h.service.store.Create(newTask)
	// Асинхронное логирование
	h.service.logChan <- "Создана новая задача: " + createdTask.Title
	// Формирование ответа
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdTask)
}

// Асинхронный логгер
func Logger(logChan <-chan string) {
	for entry := range logChan {
		log.Printf(" [ЛОГ] %s", entry)
	}
	log.Println(" Логгер остановлен")
}

func main() {
	// Инициализация компонентов
	store := NewTaskStorage()
	logChan := make(chan string, 100)
	service := NewTaskService(store, logChan)
	handler := NewTaskHandler(service)
	// Запуск асинхронного логгера
	go Logger(logChan)
	// Настройка маршрутизатора
	mux := http.NewServeMux()
	mux.HandleFunc("GET /tasks", handler.GetTasks)
	mux.HandleFunc("GET /tasks/{id}", handler.GetTaskByID)
	mux.HandleFunc("POST /tasks", handler.CreateTask)
	// Конфигурация HTTP-сервера
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	// Канал для сигналов ОС
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	// Запуск сервера в горутине
	go func() {
		log.Println(" Сервер запущен на http://localhost:8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка сервера: %v", err)
		}
	}()
	// Ожидание сигнала завершения
	<-stop
	log.Println("\n Получен сигнал завершения")
	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Ошибка завершения: %v", err)
	}
	// Закрытие канала логов после завершения работы
	close(logChan)
	log.Println(" Сервер корректно остановлен")
}
