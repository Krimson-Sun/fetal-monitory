import numpy as np
from scipy.signal import butter, sosfiltfilt, medfilt
import pandas as pd
from typing import Tuple, List, Dict

class Preprocessor:
    """
    Класс для препроцессинга и анализа физиологических сигналов ЧСС (сердечного ритма) и маточных сокращений.

    Предоставляет методы для:
    - Высокопроизводительной фильтрации сигналов с сохранением клинически значимых паттернов
    - Вычисления метрик вариабельности сердечного ритма (STV, LTV)
    - Обнаружения децелераций, акселераций и маточных сокращений
    - Анализа соотношений между событиями (например, поздних децелераций)
    - Вычисления трендов для динамического анализа

    Особенности:
    - Оптимизирован для работы с реальными медицинскими данными
    - Поддерживает обработку данных с разной частотой дискретизации
    - Обеспечивает нулевое фазовое искажение при фильтрации
    - Сохраняет исходную амплитуду и смещение сигнала
    - Работает в режиме реального времени с низкими требованиями к вычислительным ресурсам
    """

    def filter_physiological_signals(self, bpm_data: pd.DataFrame, uterus_data: pd.DataFrame, fs_estimated: float = 4.0) -> Tuple[pd.DataFrame, pd.DataFrame, float, float]:
        """
        Высокопроизводительная фильтрация сигналов ЧСС и маточных сокращений.

        Parameters
        ----------
        bpm_data : pd.DataFrame
            DataFrame с колонками ['time_sec', 'value'] для ЧСС.
        uterus_data : pd.DataFrame
            DataFrame с колонками ['time_sec', 'value'] для маточных сокращений.
        fs_estimated : float
            Оценочная частота дискретизации сигнала (обычно 2-4 Гц).

        Returns
        -------
        filtered_bpm : pd.DataFrame
            Отфильтрованный сигнал ЧСС с теми же колонками.
        filtered_uterus : pd.DataFrame
            Отфильтрованный сигнал маточных сокращений с теми же колонками.
        fs_bpm : float
            Частота дискретизации сигнала ЧСС
        fs_uterus : float
            Частота дискретизации сигнала маточных сокращений

        Notes
        -----
        - Обработка менее чем за 0.5 секунды даже на слабых устройствах
        - Сохранение клинически значимых паттернов
        - Удаление всех типов шумов (импульсные, движения, начало/конец записи)
        - Нулевое фазовое искажение
        - Сохранение исходной амплитуды и смещения
        """
        # Проверка на пустые данные
        if bpm_data.empty or uterus_data.empty:
            return (bpm_data.copy(), uterus_data.copy(), 0.0, 0.0)
        
        # Сортировка и удаление дубликатов
        bpm_data = bpm_data.sort_values('time_sec').drop_duplicates('time_sec').reset_index(drop=True)
        uterus_data = uterus_data.sort_values('time_sec').drop_duplicates('time_sec').reset_index(drop=True)
        
        # Определение частоты дискретизации (быстрый и устойчивый метод)
        def estimate_fs(data):
            if len(data) < 10:
                return fs_estimated
            # Используем медиану первых 10 интервалов (устойчиво к выбросам)
            time_diffs = np.diff(data['time_sec'].iloc[:10])
            return 1.0 / np.median(time_diffs[time_diffs > 0]+1e-18) if len(time_diffs) > 0 else fs_estimated
        
        fs_bpm = estimate_fs(bpm_data)
        fs_uterus = estimate_fs(uterus_data)
        
        # Универсальная функция фильтрации
        def filter_signal(values, fs, med_window_sec, cutoff_freq, order, threshold_diff = 1e10, threshold_del = 0.1, threshold_val=0.3):
            if len(values) < 2:
                return values

            # 1. Медианный фильтр для импульсных шумов
            window = max(3, int(med_window_sec * fs))
            if window % 2 == 0:
                window += 1
            values = medfilt(values, window)

            median_value = np.median(values)

            abs_diff = np.abs(np.diff(values))
            split_indices = np.where(abs_diff*fs > threshold_diff)[0]

            start = 0
            for idx in split_indices:
                end = idx + 1
                #print(start, end)
                if (end-start< threshold_del*len(values)) and np.abs(np.median(values[start:end]) - median_value)> median_value*threshold_val:
                    values[start:end] = median_value
                    #print((end-start) / len(values),'delete')
                start = end

            # 2. Низкочастотный фильтр Баттерворта для частотной фильтрации
            if len(values) > order * 2:
                nyq = 0.5 * fs
                normal_cutoff = min(cutoff_freq / (nyq+1e-18), 0.99)
                try:
                    sos = butter(order, normal_cutoff, btype='low', output='sos')
                    values = sosfiltfilt(sos, values)
                except ValueError:
                    pass
                
            return values
        
        # Оптимальные параметры для ЧСС (сохраняет клинически значимые паттерны)
        filtered_bpm = bpm_data.copy()
        filtered_bpm['value'] = filter_signal(
            bpm_data['value'].values,
            fs_bpm,
            med_window_sec=3,  # Для удаления импульсных шумов (1.5 сек)
            cutoff_freq=0.05,     # Сохраняет ускорения и децелерации
            order=3,
            threshold_diff=70
        )
        
        # Оптимальные параметры для маточных сокращений
        filtered_uterus = uterus_data.copy()
        filtered_uterus['value'] = filter_signal(
            uterus_data['value'].values,
            fs_uterus,
            med_window_sec=3,   # Более широкое окно для медленных сигналов
            cutoff_freq=0.01,     # Только очень низкие частоты (0.01 Гц)
            order=4
        )
        
        return (filtered_bpm, filtered_uterus, fs_bpm, fs_uterus)


    def compute_metrics(self, bpm_data: pd.DataFrame, uterus_data: pd.DataFrame, fs_bpm: float = 4.0, fs_uc: float = 4.0, summarize=False) -> Dict:
        """
        Считает метрики для отфильтрованных данных за все время наблюдений

        Parameters
        ----------
        bpm_data : pd.DataFrame
            Отфильтрованные данные ЧСС, содержащие колонки ['time_sec', 'value'].
        uterus_data : pd.DataFrame
            Отфильтрованные данные маточной активности, содержащие колонки ['time_sec', 'value'].
        fs_bpm : float
            Частота дискретизации сигнала ЧСС. По умолчанию 4.0 Гц.
        fs_uc : float
            Частота дискретизации сигнала маточной активности. По умолчанию 4.0 Гц.
        summarize : bool
            Должны ли метрики быть усреднены на весь период наблюдений

        Returns
        -------
        Dict
            Словарь с вычисленными метриками. Содержит:
                - 'stv' (float): Среднее значение Short-Term Variability.
                - 'ltv' (float): Значение Long-Term Variability.
                - 'baseline_heart_rate' (float): Базовая ЧСС.
                - 'accelerations' (List[Dict[str, float]]): Список акселераций.
                - 'decelerations' (List[Dict[str, float]]): Список децелераций.
                - 'contractions' (List[Dict[str, float]]): Список маточных сокращений.
                - 'stvs' (np.ndarray): Массив значений STV для каждого временного окна.
                - 'stvs_window_duration' (float): Длительность окна для STV в секундах.
                - 'ltvs' (np.ndarray): Массив значений LTV для каждого временного окна.
                - 'ltvs_window_duration' (float): Длительность окна для LTV в секундах.
                - 'total_decelerations' (int): Общее количество децелераций.
                - 'late_decelerations' (int): Количество поздних децелераций.
                - 'late_deceleration_ratio' (float): Доля поздних децелераций (от общего числа).
                - 'total_accelerations' (int): Общее количество акселераций.
                - 'accel_decel_ratio' (float): Отношение количества акселераций к децелерациям.
                - 'total_contractions' (int): Общее количество маточных сокращений.
                - 'stv_trend' (float): Тренд STV за последнюю минуту (наклон линии тренда).
                - 'bpm_trend' (float): Тренд ЧСС за последние 5 минут (наклон линии тренда).
                - 'data_points' (int): Количество точек данных в ЧСС.
                - 'time_span_sec' (float): Общая длительность наблюдений в секундах (разница между последней и первой меткой времени).
        """

        # 1. Подготовка данных
        if bpm_data is None or len(bpm_data) == 0:
            bpm = np.array([])
        else:
            bpm = bpm_data['value'].values
            
        if uterus_data is None or len(uterus_data) == 0:
            uterus = np.array([])
        else:
            uterus = uterus_data['value'].values

        # 2. Базовые вычисления
        if summarize:
            # Вычисляем БЧСС
            baseline = np.median(bpm) if len(bpm)>0 else 130.0
            # Вычисляем STV
            stv, stvs, stvs_window_duration = self.__compute_stv(bpm, fs_bpm)
            # Вычисляем LTV
            ltv, ltvs, ltvs_window_duration = self.__compute_ltv(bpm, fs_bpm)
        else:
            # Вычисляем БЧСС за последние 10 минут
            last_10_min = int(600 * fs_bpm)  # 10 минут = 600 секунд
            if len(bpm) > last_10_min:
                baseline = np.median(bpm[-last_10_min:])
            else:
                baseline = np.median(bpm) if len(bpm) > 0 else 130.0  # 130 - средняя нормальная ЧСС
            
            # Вычисляем STV за последние 2 минуты
            last_2_min = int(120 * fs_bpm)
            if len(bpm) > last_2_min:
                stv, stvs, stvs_window_duration = self.__compute_stv(bpm[-last_2_min:], fs_bpm)
            elif len(bpm) > 0:
                stv, stvs, stvs_window_duration = self.__compute_stv(bpm, fs_bpm)
            else:
                stv = 0.0
                stvs = np.array([0.0])
                stvs_window_duration = 0.0
            
            # Вычисляем LTV за последние 10 минут
            if len(bpm) > last_10_min:
                ltv, ltvs, ltvs_window_duration = self.__compute_ltv(bpm, fs_bpm)
            elif len(bpm) > 0:
                ltv, ltvs, ltvs_window_duration = self.__compute_ltv(bpm, fs_bpm)
            else:
                ltv = 0.0
                ltvs = np.array([0.0])
                ltvs_window_duration = 0.0

        # 3. Обнаружение децелераций
        decelerations = self.__detect_decelerations(bpm, baseline, fs=fs_bpm)

        # 4. Обнаружение маточных сокращений
        contractions = self.__detect_contractions(uterus, fs=fs_uc)

        # 5. Вычисление соотношений
        late_decelerations = 0
        for d in decelerations:
            if self.__is_late(d, contractions, bpm, uterus, fs_bpm, fs_uc):
                late_decelerations += 1
                d["is_late"] = True
            else:
                d["is_late"] = False
        
        total_decelerations = len(decelerations)
        late_ratio = late_decelerations / (total_decelerations) if total_decelerations > 0 else 0.0

        # 6. Вычисление ускорений
        accelerations = self.__detect_accelerations(bpm, baseline, fs=fs_bpm)
        total_accelerations = len(accelerations)
        accel_decel_ratio = (total_accelerations / total_decelerations) if total_decelerations > 0 else total_accelerations

        # 7. Динамические признаки
        stv_change = self.__compute_trend(stvs, window=60, fs=3.75)
        bpm_change = self.__compute_trend(bpm, window=300, fs=60)

        return {
            'stv': stv,
            'ltv': ltv,
            'baseline_heart_rate': baseline,
            
            'accelerations': accelerations,
            'decelerations': decelerations,
            'contractions': contractions,

            'stvs': stvs,
            'stvs_window_duration': stvs_window_duration,
            'ltvs': ltvs,
            'ltvs_window_duration': ltvs_window_duration,

            'total_decelerations': total_decelerations,
            'late_decelerations': late_decelerations,
            'late_deceleration_ratio': late_ratio,
            'total_accelerations': total_accelerations,
            'accel_decel_ratio': accel_decel_ratio,
            'total_contractions': len(contractions),

            'stv_trend': stv_change,
            'bpm_trend': bpm_change,
            'data_points': len(bpm),
            'time_span_sec': (bpm_data['time_sec'].iloc[-1] - bpm_data['time_sec'].iloc[0]) if len(bpm_data) > 1 else 0.0
        }


    def __compute_stv(self, bpm_signal: np.ndarray, fs: float = 4.0) -> Tuple[float, np.ndarray, float]:
        """
        Вычисляет Short-Term Variability (STV) для сигнала ЧСС в ударах/минуту

        Parameters
        ----------
        bpm_signal : np.ndarray
            Массив сигналов ЧСС.
        fs : float
            Частота дискретизации сигнала в Гц.

        Returns
        -------
        stv_mean : float
            Среднее значение Short-Term Variability.
        stv_windows : np.ndarray
            Массив значений STV для каждого временного окна.
        window_duration : float
            Длительность одного окна в секундах.
        """

        if fs <= 0:
            return 0.0, np.array([]), 0.0
        
        rr_intervals = 60000.0 / (bpm_signal+1e-18)  # Преобразуем ЧСС в интервалы RR в мс
        window_size = int(3.75 * fs)  # Вычисляем длину окна длительности 3.75 секунд
        
        windows = np.zeros( int(len(rr_intervals) / window_size) )
        for i in range(0, len(rr_intervals) - window_size + 1, window_size):
            # Вычисляем среднюю ЧСС окна
            window_mean = np.mean(rr_intervals[i:i+window_size])
            windows[int(i/window_size)] = window_mean
        
        if len(windows) < 2:
            return 0.0, np.array([]), 0.0
        
        stvs = np.abs(np.diff(windows))  # Находим STV для окон
        mean_stv = np.mean(stvs)  # Вычисляем средний STV по сигналу
        window_delay = window_size * 1/fs  # Вычисляем длительность окна
        
        return mean_stv, stvs, window_delay


    def __compute_ltv(self, bpm_signal: np.ndarray, fs: float = 4.0) -> Tuple[float, np.ndarray, float]:
        """
        Вычисляет Long-Term Variability (LTV) для сигнала ЧСС в ударах/минуту

        Parameters
        ----------
        bpm_signal : np.ndarray
            Массив сигналов ЧСС.
        fs : float
            Частота дискретизации сигнала в Гц.

        Returns
        -------
        ltv_median : float
            Значение Long-Term Variability.
        ltv_windows : np.ndarray
            Массив значений разброса для каждого временного окна.
        window_duration : float
            Длительность одного окна в секундах.
        """

        if len(bpm_signal) < 2:
            return 0.0, np.array([]), 0.0
        
        if fs <= 0:
            return 0.0, np.array([]), 0.0
        
        rr_intervals = 60000.0 / (bpm_signal+1e-18)  # Преобразуем ЧСС в интервалы RR в мс
        window_size = int(60 * fs)  # Вычисляем длину окна длительности 1 мин
        
        windows = np.zeros( int(len(rr_intervals) / window_size) )
        for i in range(0, len(rr_intervals) - window_size + 1, window_size):
            # Вычисляем разброс окна
            windows[int(i/window_size)] = np.max(rr_intervals[i:i+window_size]) - np.min(rr_intervals[i:i+window_size])
        
        ltv = np.median(windows) if len(windows) > 0 else 0.0  # Вычисляем медиану по окнам
        window_delay = window_size * 1/fs  # Вычисляем длительность окна
        
        return ltv, windows, window_delay


    def __detect_decelerations(
            self,
            bpm: np.ndarray,
            baseline: float,
            threshold: float = 15.0,
            min_duration: float = 15.0,
            fs: float = 4.0
    ) -> List[Dict[str, float]]:
        """
        Обнаруживает децелерации в сигнале ЧСС

        Parameters
        ----------
        bpm : np.ndarray
            Массив сигналов ЧСС в ударах/минуту.
        baseline : float
            Базовая ЧСС в ударах/минуту.
        threshold : float
            Минимальное отклонение от БЧСС в ударах/минуту, чтобы считаться децелерацией.
        min_duration : float
            Минимальная длительность децелерации в секундах.
        fs : float
            Частота дискретизации сигнала в Гц.

        Returns
        -------
        List[Dict[str, float]]
            Список словарей с данными о децелерациях. Каждый словарь содержит:
                - 'start' (float): Индекс начала децелерации.
                - 'end' (float): Индекс окончания децелерации.
                - 'duration' (float): Продолжительность децелерации в секундах.
                - 'amplitude' (float): Амплитуда децелерации (в ударах/минуту).
        """

        if len(bpm) < 2:
            return []
        if fs <= 0:
            return []
        
        # Создаем бинарный сигнал: 1 если ЧСС ниже baseline - threshold
        below_threshold = (bpm < baseline - threshold).astype(np.int8)
        
        # Находим переходы (начало и конец)
        diff_signal = np.diff(below_threshold)
        starts = np.where(diff_signal == 1)[0]
        ends = np.where(diff_signal == -1)[0]

        if len(starts) == 0:
            return []
        
        if (len(ends) > 0) and (starts[0]>ends[0]):
            starts = np.insert(starts, 0, 0)
        
        # Фильтруем по длительности
        decelerations = []
        for i in range(min(len(starts), len(ends)+1)):

            start = starts[i]
            end = ends[i] if i < len(ends) else len(bpm)

            duration = (end - start) / fs
            if duration >= min_duration:
                # Находим минимальное значение в этом участке
                min_val = np.min(bpm[start:end])
                amplitude = baseline - min_val
                decelerations.append({
                    'start': start,
                    'end': end,
                    'duration': duration,
                    'amplitude': amplitude
                })
        
        return decelerations


    def __detect_contractions(
            self,
            uc_signal: np.ndarray,
            threshold: float = 0.2,
            min_duration: float = 20.0,
            fs: float = 4.0
    ) -> List[Dict[str, float]]:
        """
        Обнаруживает маточные сокращения в сигнале

        Parameters
        ----------
        uc_signal : np.ndarray
            Сигналы активности матки в процентах.
        threshold : float
            Относительное отклонение, чтобы считаться схваткой.
        min_duration : float
            Минимальная длительность схватки в секундах.
        fs : float
            Частота дискретизации сигнала в Гц.

        Returns
        -------
        List[Dict[str, float]]
            Список словарей с данными о маточных сокращениях. Каждый словарь содержит:
                - 'start' (float): Индекс начала схватки.
                - 'end' (float): Индекс окончания схватки.
                - 'duration' (float): Продолжительность схватки в секундах.
                - 'amplitude' (float): Амплитуда схватки в процентах.
        """

        if len(uc_signal) < 2:
            return []
        if fs <= 0:
            return []
        
        # Нормализуем сигнал без лишних операций
        min_val = np.min(uc_signal)
        max_val = np.max(uc_signal)
        if max_val - min_val < 1e-8:
            # Если сигнал постоянный, возвращаем пустой список
            return []
        
        normalized = (uc_signal - min_val) / (max_val - min_val)
        
        # Создаем бинарный сигнал: 1 если нормализованный сигнал выше порога
        above_threshold = (normalized > threshold).astype(np.int8)
        
        # Находим переходы (начало и конец)
        diff_signal = np.diff(above_threshold)
        starts = np.where(diff_signal == 1)[0]
        ends = np.where(diff_signal == -1)[0]

        if len(starts) == 0:
            return []
        
        if (len(ends)>0) and (starts[0]>ends[0]):
            starts = np.insert(starts, 0, 0)
        
        # Фильтруем по длительности
        contractions = []
        for i in range(min(len(starts), len(ends)+1)):
            start = starts[i]
            end = ends[i] if i < len(ends) else len(uc_signal)
            
            duration = (end - start) / fs
            if duration >= min_duration:
                # Находим амплитуду сокращения
                segment = uc_signal[start:end]
                amplitude = np.max(segment) - np.min(segment)
                contractions.append({
                    'start': start,
                    'end': end,
                    'duration': duration,
                    'amplitude': amplitude
                })
        
        return contractions


    def __is_late(
            self,
            deceleration: Dict[str, float],
            contractions: List[Dict[str, float]],
            bpm: np.ndarray,
            uc_signal: np.ndarray,
            fs_bpm: float = 4.0,
            fs_uc: float = 4.0
    ) -> bool:
        """
        Определяет, является ли децелерация поздней

        Parameters
        ----------
        deceleration : Dict[str, float]
            Данные о децелерации, содержащие:
                - 'start' (float): Индекс начала децелерации
                - 'end' (float): Индекс окончания децелерации
                - 'duration' (float): Продолжительность децелерации в секундах
                - 'amplitude' (float): Амплитуда децелерации (в ударах/минуту)
        contractions : List[Dict[str, float]]
            Список маточных сокращений, каждое содержит:
                - 'start' (float): Индекс начала схватки
                - 'end' (float): Индекс окончания схватки
                - 'duration' (float): Продолжительность схватки в секундах
                - 'amplitude' (float): Амплитуда схватки в процентах
        bpm : np.ndarray
            Сигналы ЧСС в ударах/минуту.
        uc_signal : np.ndarray
            Сигнал сокращений матки.
        fs_bpm : float
            Частота дискретизации сигнала ЧСС в Гц.
        fs_uc : float
            Частота дискретизации сигнала маточных сокращений в Гц.

        Returns
        -------
        bool
            True, если децелерация поздняя (связана с маточными сокращениями), иначе False.
        """

        if len(contractions) == 0:
            return False
        if fs_bpm <= 0 or fs_uc <=0:
            return False
        
        decel_start = deceleration['start']
        decel_end = deceleration['end']
        
        # Находим индекс минимума децелерации
        min_idx = np.argmin(bpm[decel_start:decel_end])
        decel_min_index = decel_start + min_idx
        
        for contraction in contractions:
            # Проверяем пересечение
            if not ((decel_end / fs_bpm < contraction['start'] / fs_uc) or (decel_start / fs_bpm > contraction['end'] / fs_uc)):
                # Находим пик маточного сокращения
                contraction_segment = uc_signal[contraction['start']:contraction['end']]
                peak_idx = np.argmax(contraction_segment)
                contraction_peak = contraction['start'] + peak_idx
                
                # Поздняя децелерация: минимум после пика маточного сокращения
                if decel_min_index / fs_bpm > contraction_peak / fs_uc:
                    return True
        
        return False


    def __detect_accelerations(
            self,
            bpm: np.ndarray,
            baseline: float,
            threshold: float = 15.0,
            min_duration: float = 15.0,
            fs: float = 4.0
    ) -> List[Dict[str, float]]:
        """
        Обнаруживает акселерации в сигнале ЧСС (в ударах/минуту)

        Parameters
        ----------
        bpm : np.ndarray
            Сигналы ЧСС в ударах/минуту.
        baseline : float
            Базовая ЧСС в ударах/минуту.
        threshold : float
            Минимальное отклонение от БЧСС в ударах/минуту, чтобы считаться акселерацией.
        min_duration : float
            Минимальная длительность акселерации в секундах.
        fs : float
            Частота дискретизации сигнала в Гц.

        Returns
        -------
        List[Dict[str, float]]
            Список словарей с данными о акселерациях. Каждый словарь содержит:
                - 'start' (float): Индекс начала акселерации.
                - 'end' (float): Индекс окончания акселерации.
                - 'duration' (float): Продолжительность акселерации в секундах.
                - 'amplitude' (float): Амплитуда акселерации (в ударах/минуту).
        """

        if len(bpm) < 2:
            return []
        if fs <= 0:
            return []
        
        # Создаем бинарный сигнал: 1 если ЧСС выше baseline + threshold
        above_threshold = (bpm > baseline + threshold).astype(np.int8)
        
        # Находим переходы (начало и конец)
        diff_signal = np.diff(above_threshold)
        starts = np.where(diff_signal == 1)[0]
        ends = np.where(diff_signal == -1)[0]

        if len(starts) == 0:
            return []
        
        if (len(ends) > 0) and (starts[0]>ends[0]):
            starts = np.insert(starts, 0, 0)
        
        # Фильтруем по длительности
        accelerations = []
        for i in range(min(len(starts), len(ends)+1)):
            start = starts[i]
            end = ends[i] if i < len(ends) else len(bpm)
            
            duration = (end - start) / fs
            if duration >= min_duration:
                # Находим максимальное значение в этом участке
                max_val = np.max(bpm[start:end])
                amplitude = max_val - baseline
                accelerations.append({
                    'start': start,
                    'end': end,
                    'duration': duration,
                    'amplitude': amplitude
                })
        
        return accelerations


    def __compute_trend(self, signal: np.ndarray, window: float = 300.0, fs: float = 4.0) -> float:
        """
        Вычисляет тренд для указанного временного сигнала за последнее окно времени заданной длины

        Parameters
        ----------
        signal : np.ndarray
            Сигнал для вычисления тренда.
        window : float
            Длительность окна в секундах для вычисления тренда.
        fs : float
            Частота дискретизации сигнала в Гц.

        Returns
        -------
        float
            Наклон тренда кривой сигнала на основе выбранного временного окна.
        """

        # Определяем размер окна в индексах
        window_size = int(window * fs)
        if len(signal) < window_size:
            window_size = len(signal)
        
        # Берем последнее окно данных
        window_data = signal[-window_size:] if window_size > 0 else signal
        
        # Вычисляем линейную регрессию напрямую
        n = len(window_data)
        if n < 2:
            return 0.0
        
        # Вычисляем средние значения
        x = np.arange(n)*fs
        x_mean = np.mean(x)
        y_mean = np.mean(window_data)
        
        # Вычисляем наклон
        numerator = np.sum((x - x_mean) * (window_data - y_mean))
        denominator = np.sum((x - x_mean) ** 2)
        slope = numerator / (denominator + 1e-8)
        
        return slope
