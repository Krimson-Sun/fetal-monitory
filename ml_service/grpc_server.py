import asyncio
import logging
from concurrent import futures
from typing import Dict
import grpc

# Импорты для gRPC
import ml_service_pb2
import ml_service_pb2_grpc

from collector import FeatureCollector
from inference import ClassifierModel

# Настройка логирования
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class MLService(ml_service_pb2_grpc.MLServiceServicer):
    """
    gRPC сервис для ML инференса.
    
    Принимает признаки из feature_extractor, накапливает их и выполняет
    предсказание состояния плода с помощью CatBoost модели.
    """
    
    def __init__(self, model_type: str = "catboost", weights_folder: str = "./weights"):
        """
        Инициализирует ML сервис.
        
        Parameters
        ----------
        model_type : str
            Тип модели (по умолчанию "catboost")
        weights_folder : str
            Путь к папке с весами модели
        """
        self.model = ClassifierModel(model_type, weights_folder)
        # Словарь коллекторов для каждой сессии
        self.collectors: Dict[str, FeatureCollector] = {}
        # Последние предсказания для каждой сессии
        self.last_predictions: Dict[str, float] = {}
        logger.info(f"MLService initialized with model_type={model_type}, weights_folder={weights_folder}")
    
    def _get_or_create_collector(self, session_id: str) -> FeatureCollector:
        """Получает или создает коллектор для указанной сессии"""
        if session_id not in self.collectors:
            self.collectors[session_id] = FeatureCollector()
            logger.info(f"Created new collector for session: {session_id}")
        return self.collectors[session_id]
    
    async def PredictFromFeatures(self, request, context) -> object:
        """
        Выполняет предсказание на основе признаков.
        
        Args:
            request: PredictRequest с признаками
            context: gRPC context
            
        Returns:
            PredictResponse с предсказанием
        """
        try:
            session_id = request.session_id
            logger.info(f"Received predict request for session: {session_id}")
            
            # Получаем коллектор для сессии
            collector = self._get_or_create_collector(session_id)
            
            # Добавляем новые признаки в коллектор
            features = {
                'stv': request.stv,
                'ltv': request.ltv,
                'baseline_heart_rate': request.baseline_heart_rate,
                'total_decelerations': request.total_decelerations,
                'late_decelerations': request.late_decelerations,
                'late_deceleration_ratio': request.late_deceleration_ratio,
                'total_accelerations': request.total_accelerations,
                'accel_decel_ratio': request.accel_decel_ratio,
                'total_contractions': request.total_contractions,
                'stv_trend': request.stv_trend,
                'bpm_trend': request.bpm_trend,
                'data_points': request.data_points,
                'time_span_sec': request.time_span_sec
            }
            
            collector.update_features(features)
            
            # Проверяем, достаточно ли данных для предсказания
            # Для начала требуем минимум 1 семпл (можно увеличить если нужна история)
            min_samples = 1
            
            if not collector.has_enough_data(min_samples):
                # Недостаточно данных - возвращаем последний предикт или 0.0
                prediction = self.last_predictions.get(session_id, 0.0)
                
                response = ml_service_pb2.PredictResponse(
                    session_id=session_id,
                    batch_ts_ms=request.batch_ts_ms,
                    prediction=prediction,
                    status="processing",
                    message=f"Accumulating data: {collector.get_count()}/{min_samples} samples",
                    has_enough_data=False
                )
                
                logger.info(f"Not enough data for session {session_id}: {collector.get_count()}/{min_samples}")
                return response
            
            # Получаем последние признаки для инференса
            features_df = collector.get_latest_features()
            
            if features_df is None or len(features_df) == 0:
                # Нет данных - возвращаем последний предикт
                prediction = self.last_predictions.get(session_id, 0.0)
                
                response = ml_service_pb2.PredictResponse(
                    session_id=session_id,
                    batch_ts_ms=request.batch_ts_ms,
                    prediction=prediction,
                    status="error",
                    message="No features available",
                    has_enough_data=False
                )
                
                return response
            
            # Выполняем инференс (может быть долгим)
            logger.info(f"Running inference for session {session_id}")
            
            # Запускаем инференс в отдельном потоке чтобы не блокировать event loop
            loop = asyncio.get_event_loop()
            prediction_array = await loop.run_in_executor(None, self.model, features_df)
            
            # prediction_array это numpy array с вероятностями для каждого семпла
            # Берем первое (и единственное) значение
            prediction = float(prediction_array[0])
            
            # Сохраняем последний предикт
            self.last_predictions[session_id] = prediction
            
            response = ml_service_pb2.PredictResponse(
                session_id=session_id,
                batch_ts_ms=request.batch_ts_ms,
                prediction=prediction,
                status="success",
                message=f"Prediction successful (samples: {collector.get_count()})",
                has_enough_data=True
            )
            
            logger.info(f"Prediction for session {session_id}: {prediction:.4f}")
            return response
            
        except Exception as e:
            logger.error(f"Error during prediction: {e}", exc_info=True)
            
            # Возвращаем последний известный предикт при ошибке
            prediction = self.last_predictions.get(request.session_id, 0.0)
            
            response = ml_service_pb2.PredictResponse(
                session_id=request.session_id,
                batch_ts_ms=request.batch_ts_ms,
                prediction=prediction,
                status="error",
                message=f"Error during prediction: {str(e)}",
                has_enough_data=False
            )
            
            return response
    
    async def ResetCollector(self, request, context) -> object:
        """
        Сбрасывает коллектор для указанной сессии.
        
        Args:
            request: ResetCollectorRequest
            context: gRPC context
            
        Returns:
            ResetCollectorResponse
        """
        try:
            session_id = request.session_id
            
            # Создаем новый коллектор (это эквивалентно сбросу)
            self.collectors[session_id] = FeatureCollector()
            # Сбрасываем последний предикт
            self.last_predictions[session_id] = 0.0
            
            logger.info(f"Reset collector for session: {session_id}")
            
            response = ml_service_pb2.ResetCollectorResponse(
                success=True,
                message=f"Collector reset successfully for session: {session_id}"
            )
            
            return response
            
        except Exception as e:
            logger.error(f"Error resetting collector: {e}")
            
            response = ml_service_pb2.ResetCollectorResponse(
                success=False,
                message=f"Error resetting collector: {str(e)}"
            )
            
            return response

async def serve():
    """Запускает gRPC сервер"""
    server = grpc.aio.server(futures.ThreadPoolExecutor(max_workers=10))
    
    service = MLService(model_type="catboost", weights_folder="./weights")
    ml_service_pb2_grpc.add_MLServiceServicer_to_server(service, server)
    
    listen_addr = '[::]:50053'
    server.add_insecure_port(listen_addr)
    
    logger.info(f"Starting ML gRPC server on {listen_addr}")
    await server.start()
    
    try:
        await server.wait_for_termination()
    except KeyboardInterrupt:
        logger.info("Shutting down ML gRPC server...")
        await server.stop(5)

if __name__ == '__main__':
    asyncio.run(serve())

