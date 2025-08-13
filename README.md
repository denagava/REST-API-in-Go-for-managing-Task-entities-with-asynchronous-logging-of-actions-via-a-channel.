# Task Management API

REST API для управления задачами с асинхронным логированием

## Требования
- Go 1.20+

## Запуск приложения
```bash

   Создать задачу
POST /tasks
curl -X POST http://localhost:8080/tasks -H "Content-Type: application/json" -d '{
  "title": "Изучить Go",
  "completed": false
}'

# Получить все задачи
GET /tasks
curl http://localhost:8080/tasks

# Получить задачу по ID
GET /tasks/{id}
curl http://localhost:8080/tasks/1

# Фильтрация по статусу
POST /tasks
curl http://localhost:8080/tasks?completed=true
