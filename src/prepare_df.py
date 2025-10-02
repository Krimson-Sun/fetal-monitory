from utils import filter_utils as filters, data_utils as du


if __name__ == "__main__":
    TO_SAVE = True
    DATASET_DIR = "/home/be2r/hackathons/fetal-monitory/large_data/raw/ИТЭЛМА_ЛЦТ"
    TARGET_CLASSES = ["hypoxia", "regular"]
    SENSOR_TYPES = ["bpm", "uterus"]

    SENSOR_CONFIGS = {
        "bpm": {
            "med_window_sec": 3,
            "cutoff_freq": 0.05,
            "order": 3,
            "threshold_diff": 70,
        },
        "uterus": {
            "med_window_sec": 3,
            "cutoff_freq": 0.01,
            "order": 4,
            "threshold_diff": None,
        },
    }

    # загружаем сырые данные, на одного пациента несколько кусков КТГ
    df = du.load_raw_dataset(DATASET_DIR)

    # трансформируем данные в формат uid -> одно полное КТГ
    df["patient_uid"] = df["class"] + "_" + df["patient_id"].astype(str)
    patient_uids = list(df.patient_uid.unique())

    signal_df = du.form_signal_dataset(patient_uids)

    filtered_signal_df = (
        signal_df.groupby(["patient_uid", "sensor_type"], group_keys=False)
        .apply(
            lambda g: (
                filters.filter_signal_df(
                    g[["time_sec", "value"]],
                    fs_estimated=4.0,
                    **SENSOR_CONFIGS.get(g.name[1], SENSOR_CONFIGS["uterus"])
                ).assign(patient_uid=g.name[0], sensor_type=g.name[1])
            )
        )
        .reset_index(drop=True)
    )

    if TO_SAVE:
        filtered_signal_df.to_pickle("filtered_signals_df.pkl")
