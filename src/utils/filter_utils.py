import numpy as np
import pandas as pd
from scipy.signal import butter, medfilt, sosfiltfilt


def prepare_signal(data: pd.DataFrame) -> pd.DataFrame:
    """Сортировка по времени и удаление дубликатов."""
    if data.empty:
        return data.copy()
    return (
        data.sort_values("time_sec").drop_duplicates("time_sec").reset_index(drop=True)
    )


def estimate_fs(data: pd.DataFrame, fs_fallback: float = 4.0) -> float:
    """Оценка частоты дискретизации на основе первых 10 отсчётов."""
    if len(data) < 10:
        return fs_fallback
    time_diffs = np.diff(data["time_sec"].iloc[:10])
    valid_diffs = time_diffs[time_diffs > 0]
    return 1.0 / np.median(valid_diffs) if len(valid_diffs) > 0 else fs_fallback


def filter_signal(
    values: np.ndarray,
    fs: float,
    med_window_sec: float,
    cutoff_freq: float,
    order: int,
    threshold_diff: float = 1e10,
    threshold_del: float = 0.1,
    threshold_val: float = 0.3,
) -> np.ndarray:
    """
    Универсальная фильтрация:
    1. Медианный фильтр для импульсных шумов.
    2. Подавление локальных сегментов с сильными артефактами.
    3. Низкочастотный фильтр Баттерворта.
    """
    if len(values) < 2:
        return values

    # --- медианный фильтр ---
    window = max(3, int(med_window_sec * fs))
    if window % 2 == 0:
        window += 1
    values = medfilt(values, window)

    # --- подавление аномальных сегментов ---
    median_value = np.median(values)
    abs_diff = np.abs(np.diff(values))
    split_indices = np.where(abs_diff * fs > threshold_diff)[0]

    start = 0
    for idx in split_indices:
        end = idx + 1
        if (end - start < threshold_del * len(values)) and np.abs(
            np.median(values[start:end]) - median_value
        ) > median_value * threshold_val:
            values[start:end] = median_value
        start = end

    # --- фильтр Баттерворта ---
    if len(values) > order * 2:
        nyq = 0.5 * fs
        normal_cutoff = min(cutoff_freq / nyq, 0.99)
        sos = butter(order, normal_cutoff, btype="low", output="sos")
        values = sosfiltfilt(sos, values)

    return values


def filter_signal_df(
    data: pd.DataFrame,
    fs_estimated: float = 4.0,
    *,
    med_window_sec: int = 3,
    cutoff_freq: float = 0.05,
    order: int = 3,
    threshold_diff: float | None = None,
) -> pd.DataFrame:
    """
    Фильтрация физиологического сигнала (полный пайплайн).
    """
    data = prepare_signal(data)
    if data.empty:
        return data.copy()

    fs = estimate_fs(data, fs_fallback=fs_estimated)

    filtered = data.copy()
    filtered["value"] = filter_signal(
        data["value"].values,
        fs=fs,
        med_window_sec=med_window_sec,
        cutoff_freq=cutoff_freq,
        order=order,
        threshold_diff=threshold_diff if threshold_diff else 1e10,
    )
    return filtered


def filter_physiological_signals(
    bpm_data: pd.DataFrame,
    uterus_data: pd.DataFrame,
    fs_estimated: float = 4.0,
) -> tuple[pd.DataFrame, pd.DataFrame]:
    """
    Фильтрация сигналов ЧСС и маточных сокращений.
    """
    filtered_bpm = filter_signal_df(
        bpm_data,
        fs_estimated=fs_estimated,
        med_window_sec=3,
        cutoff_freq=0.05,
        order=3,
        threshold_diff=70,
    )

    filtered_uterus = filter_signal_df(
        uterus_data,
        fs_estimated=fs_estimated,
        med_window_sec=3,
        cutoff_freq=0.01,
        order=4,
    )

    return filtered_bpm, filtered_uterus
