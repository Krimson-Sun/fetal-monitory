import os
import time
import os.path as osp
from glob import glob
from typing import List, Dict, Union

import numpy as np
import pandas as pd


def load_raw_dataset(
    dataset_dir: str, target_classes: List[str], sensor_types: List[str]
) -> pd.DataFrame:
    """
    Загружает данные с датчиков всех пациентов в один DataFrame в длинном формате.

    Параметры:
        dataset_dir: путь к корневой директории с данными
        target_classes: список целевых классов для загрузки
        sensor_types: список типов датчиков для загрузки

    Функция обходит структуру папок:
      dataset_dir/класс/patient_id/тип_датчика/файлы.csv

    Возвращает DataFrame со столбцами:
      - patient_id: идентификатор пациента (папка)
      - class: класс (hypoxia, regular)
      - sensor_type: тип датчика (bpm, uterus)
      - file_id: имя файла без расширения
      - time_index: временной индекс
      - time_sec: время в секундах
      - value: значения с датчика
    """
    records = []

    for target_class in target_classes:
        class_dir = osp.join(dataset_dir, target_class)
        if not osp.exists(class_dir):
            continue

        for folder_id in os.listdir(class_dir):
            folder_path = osp.join(class_dir, folder_id)
            if not osp.isdir(folder_path):
                continue

            for sensor_type in sensor_types:
                sensor_dir = osp.join(folder_path, sensor_type)
                if not osp.exists(sensor_dir):
                    continue

                csv_files = glob(osp.join(sensor_dir, "*.csv"))
                for file_idx, file in enumerate(csv_files):
                    df = pd.read_csv(file)
                    df = df.reset_index().rename(columns={"index": "time_index"})
                    df["patient_id"] = folder_id
                    df["class"] = target_class
                    df["sensor_type"] = sensor_type
                    df["file_id"] = osp.splitext(osp.basename(file))[0]
                    records.append(df)

    full_df = pd.concat(records, ignore_index=True)
    return full_df


def glue_patient_data(
    patient_uid: str, sensor_type: str, dataset_dir: str, normalize_first: bool = True
) -> pd.DataFrame:

    # получаем данные сенсора (sensor_type) для пациента (patient_uid)
    target_class, folder_id = patient_uid.split("_")  # hypoxia_31 -> hypoxia, 31
    csv_dir = osp.join(dataset_dir, target_class, folder_id, sensor_type)

    filenames = [f for f in os.listdir(csv_dir) if f.endswith(".csv")]
    filenames = sorted(filenames, key=lambda x: int(x.split("-")[1].split("_")[0]))

    if not filenames:
        return pd.DataFrame()

    parts = []
    for idx, fname in enumerate(filenames):
        path = osp.join(csv_dir, fname)
        part = pd.read_csv(path)
        part = part.copy()

        if idx == 0:
            if normalize_first:
                part["time_sec"] = part["time_sec"] - part["time_sec"].min()
            parts.append(part)
            prev_max = part["time_sec"].max()
            continue
        part["time_sec"] = part["time_sec"] + prev_max

        parts.append(part)
        prev_max = part["time_sec"].max()

    glued = pd.concat(parts, ignore_index=True)
    glued = glued.sort_values("time_sec").reset_index(drop=True)
    return glued


def form_signal_dataset(patient_uids: List[str], dataset_dir: str):
    all_records = []

    for pid in patient_uids:
        for sensor in ["bpm", "uterus"]:
            df_sensor = glue_patient_data(pid, sensor, dataset_dir)

            if df_sensor.empty:
                continue
            df_long = df_sensor[["time_sec", "value"]].rename(
                columns={"value": "value"}
            )
            df_long["sensor_type"] = sensor
            df_long["patient_uid"] = pid

            all_records.append(df_long)

    result = pd.concat(all_records, ignore_index=True)
    return result
