import pandas as pd
from typing import Dict, Optional, Tuple


class Collector:
    """
    Класс для сбора и управления данными ЧСС (сердечного ритма) и маточных сокращений в режиме реального времени.

    Хранит данные в двух внутренних DataFrame:
    - 'bpm': данные ЧСС с колонками 'time_sec' (время в секундах) и 'value' (значение ЧСС в ударах/минуту)
    - 'uterus': данные маточных сокращений с колонками 'time_sec' и 'value' (амплитуда сокращений)

    Методы:
    - __init__: инициализирует пустые DataFrame для хранения данных
    - update_collector: добавляет новые данные в соответствующие DataFrame
    - get_data: возвращает текущие собранные данные в виде кортежа из двух DataFrame
    """

    def __init__(self):
        """
        Инициализирует пустые DataFrame для хранения данных ЧСС и маточных сокращений.

        Создает два DataFrame с колонками:
        - 'time_sec': временная метка в секундах
        - 'value': значение сигнала (ЧСС в ударах/минуту для bpm, амплитуда сокращений для uterus)

        Returns
        -------
        None
        """
        self.__collector = {
            "bpm": pd.DataFrame(columns=["time_sec", "value"]),
            "uterus": pd.DataFrame(columns=["time_sec", "value"]),
        }

    def update_collector(self, new_data: Dict[str, Optional[float]]) -> None:
        """
        Обновляет коллектор, добавляя новые данные в соответствующие DataFrame.

        Parameters
        ----------
        new_data : Dict[str, Optional[float]]
            Словарь с новыми данными. Ожидаемые ключи:
                - 'bpm_s' (Optional[float]): Новое значение ЧСС (в ударах/минуту). Если None, данные не добавляются.
                - 'uc_s' (Optional[float]): Новое значение маточных сокращений (амплитуда). Если None, данные не добавляются.
                - 'time_sec' (float): Временная метка в секундах, для которой предоставляются данные.

        Notes
        -----
        - Если значение 'bpm_s' или 'uc_s' равно None, соответствующие данные не добавляются.
        - Временная метка 'time_sec' всегда обязательна.
        - Данные добавляются в конец соответствующего DataFrame.

        Returns
        -------
        None
        """
        if ("bpm_s" in new_data) and new_data["bpm_s"] is not None:
            self.__collector["bpm"].loc[len(self.__collector["bpm"])] = [
                new_data["time_sec"],
                new_data["bpm_s"],
            ]
        if ("uc_s" in new_data) and new_data["uc_s"] is not None:
            self.__collector["uterus"].loc[len(self.__collector["uterus"])] = [
                new_data["time_sec"],
                new_data["uc_s"],
            ]

    def get_data(self) -> Tuple[pd.DataFrame, pd.DataFrame]:
        """
        Возвращает текущие собранные данные в виде кортежа из двух DataFrame.

        Returns
        -------
        Tuple[pd.DataFrame, pd.DataFrame]
            - Первый элемент: DataFrame с данными ЧСС, содержащий колонки:
                - 'time_sec': временные метки в секундах
                - 'value': значения ЧСС в ударах/минуту
            - Второй элемент: DataFrame с данными маточных сокращений, содержащий колонки:
                - 'time_sec': временные метки в секундах
                - 'value': амплитуда маточных сокращений
        """
        return self.__collector["bpm"], self.__collector["uterus"]
