import pandas as pd
from typing import Dict, Optional

class FeatureCollector:
    """
    Класс для накопления признаков из feature_extractor для ML модели.
    
    Накапливает фичи в DataFrame до достижения необходимого количества данных
    для инференса модели.
    """
    
    def __init__(self):
        """
        Инициализирует пустой DataFrame для накопления фич.
        
        Колонки соответствуют выходу feature_extractor и входу ML модели.
        """
        self.__features = pd.DataFrame(columns=[
            'stv', 
            'ltv', 
            'baseline_heart_rate',
            'total_decelerations',
            'late_decelerations', 
            'late_deceleration_ratio',
            'total_accelerations',
            'accel_decel_ratio',
            'total_contractions',
            'stv_trend',
            'bpm_trend',
            'data_points',
            'time_span_sec'
        ])
    
    def update_features(self, features: Dict) -> None:
        """
        Добавляет новый набор признаков в коллектор.
        
        Parameters
        ----------
        features : Dict
            Словарь с признаками из feature_extractor.
            Ожидаемые ключи соответствуют колонкам DataFrame.
        
        Returns
        -------
        None
        """
        # Создаем новую строку с признаками
        new_row = {
            'stv': features.get('stv', 0.0),
            'ltv': features.get('ltv', 0.0),
            'baseline_heart_rate': features.get('baseline_heart_rate', 0.0),
            'total_decelerations': features.get('total_decelerations', 0),
            'late_decelerations': features.get('late_decelerations', 0),
            'late_deceleration_ratio': features.get('late_deceleration_ratio', 0.0),
            'total_accelerations': features.get('total_accelerations', 0),
            'accel_decel_ratio': features.get('accel_decel_ratio', 0.0),
            'total_contractions': features.get('total_contractions', 0),
            'stv_trend': features.get('stv_trend', 0.0),
            'bpm_trend': features.get('bpm_trend', 0.0),
            'data_points': features.get('data_points', 0),
            'time_span_sec': features.get('time_span_sec', 0.0)
        }
        
        # Добавляем строку в DataFrame
        self.__features.loc[len(self.__features)] = new_row
    
    def get_features(self) -> pd.DataFrame:
        """
        Возвращает накопленные признаки.
        
        Returns
        -------
        pd.DataFrame
            DataFrame с накопленными признаками
        """
        return self.__features.copy()
    
    def has_enough_data(self, min_samples: int = 1) -> bool:
        """
        Проверяет, достаточно ли данных для инференса.
        
        Parameters
        ----------
        min_samples : int
            Минимальное количество семплов для инференса
        
        Returns
        -------
        bool
            True если данных достаточно, иначе False
        """
        return len(self.__features) >= min_samples
    
    def get_latest_features(self) -> Optional[pd.DataFrame]:
        """
        Возвращает последнюю строку с признаками для инференса.
        
        Returns
        -------
        Optional[pd.DataFrame]
            DataFrame с последней строкой признаков или None если данных нет
        """
        if len(self.__features) == 0:
            return None
        return self.__features.tail(1)
    
    def reset(self) -> None:
        """
        Очищает все накопленные данные.
        
        Returns
        -------
        None
        """
        self.__features = pd.DataFrame(columns=self.__features.columns)
    
    def get_count(self) -> int:
        """
        Возвращает количество накопленных семплов.
        
        Returns
        -------
        int
            Количество строк в DataFrame
        """
        return len(self.__features)

