import pandas as pd
import numpy as np
from sklearn.model_selection import train_test_split, StratifiedKFold, cross_val_score
from catboost import CatBoostClassifier
from sklearn.metrics import (
    roc_auc_score,
    classification_report,
    confusion_matrix,
    precision_recall_fscore_support,
    classification_report,
)


def prepare_data(features_df, test_size=0.1, random_state=42):
    X = features_df.drop(
        columns=[
            "patient_uid",
            "target",
            "accelerations",
            "decelerations",
            "contractions",
            "stvs",
            "ltvs",
        ]
    )
    y = features_df["target"]
    X_train, X_test, y_train, y_test = train_test_split(
        X, y, stratify=y, test_size=test_size, random_state=random_state
    )
    return X_train, y_train, X_test, y_test


def catboost_pipeline(features_df):
    print("Подготовка данных...")
    X, y, X_final_test, y_final_test = prepare_data(features_df)

    clfs = []
    scores = []
    auc_scores = []
    kf = StratifiedKFold(n_splits=3, shuffle=True, random_state=7575)

    for train_index, test_index in kf.split(X=X, y=y):
        X_train, X_test = X.iloc[train_index], X.iloc[test_index]
        y_train, y_test = y.iloc[train_index], y.iloc[test_index]

        clf = CatBoostClassifier(
            iterations=500,
            depth=6,
            learning_rate=0.05,
            loss_function="Logloss",
            auto_class_weights="Balanced",
            eval_metric="AUC",
            random_seed=7575,
            verbose=100,
        )

        clfs.append(clf)

        clf.fit(X_train, y_train)

        y_pred = clf.predict(X_test)
        score = np.mean(np.array(y_pred == y_test))
        scores.append(score)

        y_pred_proba = clf.predict_proba(X_test)[:, 1]
        auc_score = roc_auc_score(y_test, y_pred_proba)
        auc_scores.append(auc_score)

        print(f"fold: acc: {score:.4f}, auc: {auc_score:.4f}")

    assert len(clfs) == 3
    print(
        "mean accuracy score --",
        np.mean(scores, dtype="float16"),
        np.std(scores).round(4),
    )
    print(f"Mean AUC score: {np.mean(auc_scores):.4f} ± {np.std(auc_scores):.4f}")
    return clfs, scores, auc_scores, X_final_test, y_final_test


def predict_ensemble_proba(clfs, X_test):
    all_probas = []
    for clf in clfs:
        proba = clf.predict_proba(X_test)[:, 1]
        all_probas.append(proba)

    all_probas = np.array(all_probas)
    ensemble_proba = np.mean(all_probas, axis=0)

    return ensemble_proba


def load_model(model_weights: str):
    model = CatBoostClassifier()
    model.load_model(model_weights)
    return model


def calculate_metrics(y_true, y_pred_proba, threshold=0.3):
    y_pred = (y_pred_proba >= threshold).astype(int)

    precision, recall, f1, support = precision_recall_fscore_support(
        y_true, y_pred, labels=[0, 1], average=None
    )

    metrics_df = pd.DataFrame(
        {
            "Class 0": [precision[0], recall[0], f1[0], support[0]],
            "Class 1": [precision[1], recall[1], f1[1], support[1]],
        },
        index=["Precision", "Recall", "F1-Score", "Support"],
    )

    return metrics_df, y_pred


if __name__ == "__main__":
    features_df = pd.read_pickle(
        "/home/be2r/hackathons/fetal-monitory/large_data/final/features_df_last_hypoxia.pkl"
    )
    MODE = "train"
    if MODE == "train":
        clfs, scores, auc_scores, X_test, y_test = catboost_pipeline(features_df)
        for i, clf in enumerate(clfs):
            clf.save_model(
                f"catboost_kfold_{i}.cbm",
                format="cbm",
                export_parameters=None,
                pool=None,
            )
    else:
        clfs = [load_model(f"catboost_kfold_{i}.cbm") for i in range(3)]

    X_test.to_csv('x_test.csv')
    y_test.to_csv('y.csv')

    preds = predict_ensemble_proba(clfs, X_test)
    metrics_df, y_pred_classes = calculate_metrics(y_test, preds)

    print("Метрики по классам:")
    print(metrics_df)
    print("\n" + "=" * 50)

    print("Подробный отчет:")
    print(
        classification_report(
            y_test, y_pred_classes, target_names=["Class 0", "Class 1"]
        )
    )

    macro_precision, macro_recall, macro_f1, _ = precision_recall_fscore_support(
        y_test, y_pred_classes, average="macro"
    )

    weighted_precision, weighted_recall, weighted_f1, _ = (
        precision_recall_fscore_support(y_test, y_pred_classes, average="weighted")
    )

    print(
        f"\nMacro Average: Precision={macro_precision:.4f}, Recall={macro_recall:.4f}, F1={macro_f1:.4f}"
    )
    print(
        f"Weighted Average: Precision={weighted_precision:.4f}, Recall={weighted_recall:.4f}, F1={weighted_f1:.4f}"
    )
