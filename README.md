# Task Management API

REST API для управления задачами с асинхронным логированием

## Требования
- Go 1.20+

## Запуск приложения
```bash

   Создать задачу
curl -X POST http://localhost:8080/tasks -H "Content-Type: application/json" -d '{
  "title": "Изучить Go",
  "completed": false
}'

# Получить все задачи
curl http://localhost:8080/tasks

# Получить задачу по ID
curl http://localhost:8080/tasks/1

# Фильтрация по статусу
curl http://localhost:8080/tasks?completed=true
