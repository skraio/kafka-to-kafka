# Kafka to Kafka ETL

## Обзор
В этом проекте реализован процесс ETL (извлечение, преобразование, загрузка)
для обработки транзакций, используя Kafka, Redis и Go. Сервис читает
сообщения из одного топика Kafka, выполняет фильтрацию, трансформацию и
дедупликацию, а затем записывает обработанные сообщения в другие топики Kafka.

Схема процесса представлена на изображении: [link](./img/kafka-to-kafka.drawio.png)

## Функционал

- **Extraction**: Чтение сообщений о транзакциях из топика Kafka.
- **Transformation**:
    - **Фильтрация**: Фильтрация сообщений на основе заданных правил.
    - **Преобразование**: Преобразование сообщений, оставляя только релевантные поля.
    - **Дедупликация**: Удаление дублирующихся сообщений с использованием Redis.
- **Loading**: Запись обработанных сообщений в целевые топики Kafka.

## Технологии
- Брокер сообщений: Kafka
- Язык программирования: Go
- Сервер: Docker и Docker-compose для развертывания и управления контейнерами
- In-memory база данных: Redis
- Формат сообщений: JSON-документы, описывающие элементы транзакций

## Конфигурация
### .env файл
В корневом каталоге должен быть файл `.env` с содержимым:
```env
REDIS_ADDR=redis:6379
REDIS_PASSWORD=password
REDIS_DB=0

BOOTSTRAP_SERVERS=kafka:29092
GROUP_ID=myGroup
OFFSET_RESET=earliest

CONSUMER_TOPIC=incoming-transactions
CONSUMER_TOPIC_STD=standard-transactions
CONSUMER_TOPIC_PRIV=privileged-transactions

CONFIG_FILENAME=config.json
```

### Конфигурация сообщений 
Файл `config.json` должен содержать следующую конфигурацию:
```json
{
  "filtering": {
    "key": "status",
    "value1": "standard",
    "value2": "privileged"
  },
  "transformation": ["id", "login", "payment"],
  "deduplication": {
    "key": "login",
    "time_span_seconds": 30
  }
}
```
#### Описание конфигураций
- `filtering`: Фильтрация сообщений на основе ключа `status`. Сообщения с
  `status` равным `standard` и `privileged` проходят фильтрацию в
  соответствующие топики Kafka.
- `transformation`: Преобразование сообщений, оставляя только ключи `id`, `login` и `payment`.
- `deduplication`: Дедупликация сообщений на основе ключа `login` в течение 30
  секунд. Если два сообщения с одинаковым значением `login` приходят в течение
  30 секунд, одно из них будет отброшено как дубликат.


## Установка и начало работы
### Предварительные требования
Установленный Docker

### Установка проекта
```bash
git clone https://github.com/skraio/kafka-to-kafka.git
cd kafka-to-kafka
```

### Запуск
```bash
docker-compose up --build
```

### Слушать логи сервиса
```bash
docker-compose logs -f app
```

### Взаимодействие с Kafka
#### Получить список топиков
```bash
docker-compose exec kafka kafka-topics --list --bootstrap-server kafka:9092
```

#### Отправка сообщений в топик Kafka
1. Откройте другой терминал и введите команду для отправки сообщений в топик `incoming-transactions`
    ```bash
    docker-compose exec kafka kafka-console-producer --broker-list kafka:9092 --topic incoming-transactions
    ```
2. Введите сообщение и нажмите Enter. \
    Пример сообщения:
    ```json
    {"id":1,"login":"user","payment":200,"status":"privileged"}
    ```

#### Чтение сообщений из топиков Kafka
Откройте другой терминал и введите команду для чтения из топика `privileged-transactions`
```bash
docker-compose exec kafka kafka-console-consumer --bootstrap-server kafka:9092 --topic privileged-transactions --from-beginning
```
Откройте другой терминал и введите команду для чтения из топика `standard-transactions`
```bash
docker-compose exec kafka kafka-console-consumer --bootstrap-server kafka:9092 --topic standard-transactions --from-beginning
```
