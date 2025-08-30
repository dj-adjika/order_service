# Order Service

Микросервис для обработки заказов с использованием Kafka, PostgreSQL и in-memory кеша.

## Архитектура

- **Kafka**: Прием сообщений о заказах
- **PostgreSQL**: Постоянное хранение данных
- **In-memory cache**: Быстрый доступ к данным
- **Web Interface**: Простой, но информативный

### Требования

- Docker и Docker Compose
- Golang 1.21+

#### Запуск

- docker-compose up -d
- go run cmd/server/main.go
- go run producer/producer.go (тестовый запрос)

