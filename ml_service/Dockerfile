FROM python:3.11-slim

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем requirements.txt из правильной папки (context = корень проекта)
COPY ml_service/requirements.txt .

# Устанавливаем зависимости
RUN pip install --no-cache-dir -r requirements.txt

# Копируем исходный код Python сервиса
COPY ml_service/ .

# Копируем протофайлы
COPY proto/ ./proto/

# Генерируем Python gRPC код из протофайлов
RUN python -m grpc_tools.protoc \
    --python_out=. \
    --grpc_python_out=. \
    --proto_path=proto/ml_service \
    proto/ml_service/ml_service.proto

# Устанавливаем переменные окружения
ENV PYTHONPATH=/app
ENV GRPC_PORT=50053

# Открываем порт
EXPOSE 50053

# Запускаем сервер
CMD ["python", "grpc_server.py"]

