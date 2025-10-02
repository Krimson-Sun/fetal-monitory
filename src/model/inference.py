import os
import os.path as osp
import pandas as pd
import numpy as np

import torch
from torch import nn
from catboost import CatBoostClassifier


class ClassifierModel(nn.Module):
    def __init__(self, model_type: str, weights_folder: str):
        self.model_type = model_type
        self.weights_folder = weights_folder
        self.weights_filepaths = [
            osp.join(weights_folder, filename)
            for filename in os.listdir(weights_folder)
            if filename.endswith(".cbm")
        ]
        self.model = self._load_model()

    def _load_single_model(self, ind: int = 0):
        model = CatBoostClassifier()
        model.load_model(self.weights_filepaths[ind])
        return model

    def _load_ansemble(self):
        return [self._load_single_model(i) for i in range(len(self.weights_filepaths))]

    def _load_model(self):
        return (
            self._load_ansemble()
            if len(self.weights_filepaths) > 1
            else self._load_single_model()
        )

    def predict_ensemble_proba(self, X_test):
        all_probas = []
        for clf in self.model:
            proba = clf.predict_proba(X_test)[:, 1]
            all_probas.append(proba)

        all_probas = np.array(all_probas)
        ensemble_proba = np.mean(all_probas, axis=0)

        return ensemble_proba

    def _prepare_data(self, df: pd.DataFrame) -> pd.DataFrame:
        if self.model_type == "catboost":
            columns_to_drop = [
                "patient_uid",
                "target",
                "accelerations",
                "decelerations",
                "contractions",
                "stvs",
                "ltvs",
                "time_span_sec"
            ]

            pred_df = df.drop(columns=columns_to_drop, error="ignore")
            return pred_df

        raise ValueError(f"{self.model_type} is not implemented yet")

    def __call__(self, x):
        x_processed = self._prepare_data(x)
        return self.predict_ensemble_proba(x_processed)


if __name__ == "__main__":
    model = ClassifierModel(
        "catboost", "/home/be2r/hackathons/fetal-monitory/weights"
    )
    breakpoint()
    X_test = pd.read_csv('x_test.csv')
    y_test = pd.read_csv('y.csv')
    preds = model.predict_ensemble_proba(X_test)
    breakpoint()
