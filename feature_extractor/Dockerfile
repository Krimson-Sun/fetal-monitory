FROM python:3.11-slim

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем requirements.txt из правильной папки (context = корень проекта)
COPY feature_extractor/requirements.txt .

# Устанавливаем зависимости
RUN pip install --no-cache-dir -r requirements.txt

# Копируем исходный код Python сервиса
COPY feature_extractor/ .

# Копируем протофайлы
COPY proto/ ./proto/

# Генерируем Python gRPC код из протофайлов
RUN python -m grpc_tools.protoc \
    --python_out=. \
    --grpc_python_out=. \
    --proto_path=proto/feature_extractor \
    proto/feature_extractor/feature_extractor.proto

# Устанавливаем переменные окружения
ENV PYTHONPATH=/app
ENV GRPC_PORT=50052

# Открываем порт
EXPOSE 50052

# Запускаем сервер
CMD ["python", "grpc_server.py"]
