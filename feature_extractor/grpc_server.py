import asyncio
import logging
from concurrent import futures
from typing import Dict, Optional, List
import grpc
import pandas as pd

# Импорты для gRPC
import feature_extractor_pb2
import feature_extractor_pb2_grpc

from collector import Collector
from preprocessor import Preprocessor

# Настройка логирования
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class FeatureExtractorService(feature_extractor_pb2_grpc.FeatureExtractorServiceServicer):
    """
    gRPC сервис для извлечения медицинских признаков из данных мониторинга плода.
    
    Принимает батчи данных ЧСС и маточных сокращений, обрабатывает их через
    Preprocessor и возвращает медицинские метрики.
    """
    
    def __init__(self):
        self.preprocessor = Preprocessor()
        # Словарь коллекторов для каждой сессии
        self.collectors: Dict[str, Collector] = {}
        logger.info("FeatureExtractorService initialized")
    
    def _get_or_create_collector(self, session_id: str) -> Collector:
        """Получает или создает коллектор для указанной сессии"""
        if session_id not in self.collectors:
            self.collectors[session_id] = Collector()
            logger.info(f"Created new collector for session: {session_id}")
        return self.collectors[session_id]
    
    def _convert_datapoints_to_dict(self, data_points) -> Dict[str, List[float]]:
        """Конвертирует список DataPoint в словарь для pandas"""
        if not data_points:
            return {'time_sec': [], 'value': []}
        
        return {
            'time_sec': [dp.time_sec for dp in data_points],
            'value': [dp.value for dp in data_points]
        }
    
    def _create_acceleration_response(self, acc: Dict) -> object:
        """Создает Acceleration response message"""
        return feature_extractor_pb2.Acceleration(
            start=acc['start'],
            end=acc['end'], 
            duration=acc['duration'],
            amplitude=acc['amplitude']
        )
    
    def _create_deceleration_response(self, dec: Dict) -> object:
        """Создает Deceleration response message"""
        return feature_extractor_pb2.Deceleration(
            start=dec['start'],
            end=dec['end'],
            duration=dec['duration'],
            amplitude=dec['amplitude'],
            is_late=dec.get('is_late', False)
        )
    
    def _create_contraction_response(self, cont: Dict) -> object:
        """Создает Contraction response message"""
        return feature_extractor_pb2.Contraction(
            start=cont['start'],
            end=cont['end'],
            duration=cont['duration'],
            amplitude=cont['amplitude']
        )
    
    def _create_datapoint_response(self, time_sec: float, value: float) -> object:
        """Создает DataPoint response message"""
        return feature_extractor_pb2.DataPoint(
            time_sec=time_sec,
            value=value
        )
    
    async def ProcessBatch(self, request, context) -> object:
        """
        Обрабатывает одиночный батч данных.
        
        Args:
            request: ProcessBatchRequest с данными батча
            context: gRPC context
            
        Returns:
            ProcessBatchResponse с обработанными метриками
        """
        try:
            session_id = request.session_id
            logger.info(f"Processing batch for session: {session_id}")
            
            # Конвертируем данные из gRPC формата
            bpm_dict = self._convert_datapoints_to_dict(request.bpm_data)
            uterus_dict = self._convert_datapoints_to_dict(request.uterus_data)
            
            # Если есть данные, добавляем их в коллектор
            collector = self._get_or_create_collector(session_id)
            
            # Добавляем точки в коллектор (симулируем онлайн режим)
            for i in range(max(len(bpm_dict['time_sec']), len(uterus_dict['time_sec']))):
                new_data = {}
                
                if i < len(bpm_dict['time_sec']):
                    new_data['bpm_s'] = bpm_dict['value'][i]
                    new_data['time_sec'] = bpm_dict['time_sec'][i]
                else:
                    new_data['bmp_s'] = None
                
                if i < len(uterus_dict['time_sec']):
                    new_data['uc_s'] = uterus_dict['value'][i]
                    if 'time_sec' not in new_data:
                        new_data['time_sec'] = uterus_dict['time_sec'][i]
                else:
                    new_data['uc_s'] = None
                
                if 'time_sec' in new_data:
                    collector.update_collector(new_data)
            
            # Получаем данные из коллектора
            bpm_raw, uterus_raw = collector.get_data()
            
            # Фильтруем данные
            bpm_filtered, uc_filtered = self.preprocessor.filter_physiological_signals(
                bpm_raw, uterus_raw, fs_estimated=4.0
            )
            
            # Вычисляем метрики
            metrics = self.preprocessor.compute_metrics(bpm_filtered, uc_filtered)
            
            # Создаем батчи отфильтрованных данных (последние 52 точки - скользящее окно)
            filtered_bpm_batch = bpm_filtered.tail(min(len(bpm_filtered), 52))
            filtered_uterus_batch = uc_filtered.tail(min(len(uc_filtered), 52))
            
            # Конвертируем в DataPoint messages
            filtered_bpm_points = [
                self._create_datapoint_response(row['time_sec'], row['value'])
                for _, row in filtered_bpm_batch.iterrows()
            ]
            
            filtered_uterus_points = [
                self._create_datapoint_response(row['time_sec'], row['value'])
                for _, row in filtered_uterus_batch.iterrows()
            ]
            
            # Конвертируем события в response messages
            accelerations = [self._create_acceleration_response(acc) for acc in metrics['accelerations']]
            decelerations = [self._create_deceleration_response(dec) for dec in metrics['decelerations']]
            contractions = [self._create_contraction_response(cont) for cont in metrics['contractions']]
            
            # Создаем ответ
            response = feature_extractor_pb2.ProcessBatchResponse(
                session_id=session_id,
                batch_ts_ms=request.batch_ts_ms,
                stv=metrics['stv'],
                ltv=metrics['ltv'],
                baseline_heart_rate=metrics['baseline_heart_rate'],
                accelerations=accelerations,
                decelerations=decelerations,
                contractions=contractions,
                stvs=list(metrics['stvs']),
                stvs_window_duration=metrics['stvs_window_duration'],
                ltvs=list(metrics['ltvs']),
                ltvs_window_duration=metrics['ltvs_window_duration'],
                total_decelerations=metrics['total_decelerations'],
                late_decelerations=metrics['late_decelerations'],
                late_deceleration_ratio=metrics['late_deceleration_ratio'],
                total_accelerations=metrics['total_accelerations'],
                accel_decel_ratio=metrics['accel_decel_ratio'],
                total_contractions=metrics['total_contractions'],
                stv_trend=metrics['stv_trend'],
                bpm_trend=metrics['bpm_trend'],
                data_points=metrics['data_points'],
                time_span_sec=metrics['time_span_sec'],
                filtered_bpm_batch=filtered_bpm_points,
                filtered_uterus_batch=filtered_uterus_points
            )
            
            logger.info(f"Successfully processed batch for session: {session_id}")
            return response
            
        except Exception as e:
            logger.error(f"Error processing batch: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Error processing batch: {str(e)}")
            return None
    
    async def ProcessBatchStream(self, request_iterator, context):
        """
        Обрабатывает стрим батчей данных.
        
        Args:
            request_iterator: Итератор ProcessBatchRequest
            context: gRPC context
            
        Yields:
            ProcessBatchResponse для каждого обработанного батча
        """
        try:
            async for request in request_iterator:
                response = await self.ProcessBatch(request, context)
                if response:
                    yield response
                    
        except Exception as e:
            logger.error(f"Error in batch stream: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Error in batch stream: {str(e)}")
    
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
            self.collectors[session_id] = Collector()
            
            logger.info(f"Reset collector for session: {session_id}")
            
            response = feature_extractor_pb2.ResetCollectorResponse(
                success=True,
                message=f"Collector reset successfully for session: {session_id}"
            )
            
            return response
            
        except Exception as e:
            logger.error(f"Error resetting collector: {e}")
            
            response = feature_extractor_pb2.ResetCollectorResponse(
                success=False,
                message=f"Error resetting collector: {str(e)}"
            )
            
            return response

async def serve():
    """Запускает gRPC сервер"""
    server = grpc.aio.server(futures.ThreadPoolExecutor(max_workers=10))
    
    service = FeatureExtractorService()
    feature_extractor_pb2_grpc.add_FeatureExtractorServiceServicer_to_server(service, server)
    
    listen_addr = '[::]:50052'
    server.add_insecure_port(listen_addr)
    
    logger.info(f"Starting gRPC server on {listen_addr}")
    await server.start()
    
    try:
        await server.wait_for_termination()
    except KeyboardInterrupt:
        logger.info("Shutting down gRPC server...")
        await server.stop(5)

if __name__ == '__main__':
    asyncio.run(serve())
