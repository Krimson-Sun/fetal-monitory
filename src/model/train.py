import time
import pandas as pd
import numpy as np
from sklearn.model_selection import train_test_split, StratifiedKFold
from catboost import CatBoostClassifier
from sklearn.metrics import (
    roc_auc_score,
    classification_report,
    confusion_matrix,
    precision_recall_fscore_support,
    classification_report,
    roc_curve, auc, precision_recall_curve
)

import matplotlib.pyplot as plt
import seaborn as sns

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
            "time_span_sec"
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
    X.to_csv("gt_X_train.csv", index=False)
    y.to_csv("gt_y_train.csv", index=False)
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

def plot_roc_auc(y_true, y_pred_proba, save_path='roc_auc_curve.png'):
    """
    Построение ROC-AUC кривой и сохранение графика
    """
    # Вычисляем ROC кривую и AUC
    fpr, tpr, thresholds = roc_curve(y_true, y_pred_proba)
    roc_auc = auc(fpr, tpr)
    
    # Настройка стиля графика
    plt.style.use('default')
    plt.figure(figsize=(10, 8))
    
    # Построение ROC кривой
    plt.plot(fpr, tpr, color='darkorange', lw=2, label=f'ROC curve (AUC = {roc_auc:.4f})')
    plt.plot([0, 1], [0, 1], color='navy', lw=2, linestyle='--', label='Random Classifier')
    
    # Настройка графика
    plt.xlim([0.0, 1.0])
    plt.ylim([0.0, 1.05])
    plt.xlabel('False Positive Rate', fontsize=12)
    plt.ylabel('True Positive Rate', fontsize=12)
    plt.title('Receiver Operating Characteristic (ROC) Curve', fontsize=14, fontweight='bold')
    plt.legend(loc="lower right")
    plt.grid(True, alpha=0.3)
    
    # Сохранение графика
    plt.savefig(save_path, dpi=300, bbox_inches='tight', facecolor='white')
    plt.show()
    
    return roc_auc, fpr, tpr

def plot_precision_recall_curve(y_true, y_pred_proba, save_path='precision_recall_curve.png'):
    """
    Дополнительно: построение Precision-Recall кривой
    """
    precision, recall, _ = precision_recall_curve(y_true, y_pred_proba)
    pr_auc = auc(recall, precision)
    
    plt.figure(figsize=(10, 8))
    plt.plot(recall, precision, color='blue', lw=2, label=f'PR curve (AUC = {pr_auc:.4f})')
    
    plt.xlim([0.0, 1.0])
    plt.ylim([0.0, 1.05])
    plt.xlabel('Recall', fontsize=12)
    plt.ylabel('Precision', fontsize=12)
    plt.title('Precision-Recall Curve', fontsize=14, fontweight='bold')
    plt.legend(loc="upper right")
    plt.grid(True, alpha=0.3)
    
    plt.savefig(save_path, dpi=300, bbox_inches='tight', facecolor='white')
    plt.show()
    
    return pr_auc


if __name__ == "__main__":
    features_df = pd.read_pickle(
        "/home/be2r/hackathons/fetal-monitory/large_data/final/features_df_last_hypoxia.pkl"
    )
    MODE = "val"
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
        X_train, y_train, X_test, y_test = prepare_data(features_df)

    X_test.to_csv('x_test.csv')
    y_test.to_csv('y.csv')
    ###
    start_time = time.time()
    preds = predict_ensemble_proba(clfs, X_test)
    end_time = time.time()
    print(f"Время предсказания на {X_test.shape} сэмплах занимает: {end_time - start_time}")
    metrics_df, y_pred_classes = calculate_metrics(y_test, preds, threshold=0.3)

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

    # Построение ROC-AUC кривой
    # Построение ROC-AUC кривой
    fpr, tpr, thresholds = roc_curve(y_test, preds)
    roc_auc = auc(fpr, tpr)

    plt.figure(figsize=(8, 6))
    plt.plot(fpr, tpr, color='darkorange', lw=2, label=f'ROC curve (AUC = {roc_auc:.4f})')
    plt.plot([0, 1], [0, 1], color='navy', lw=2, linestyle='--', label='Random')
    plt.xlim([0.0, 1.0])
    plt.ylim([0.0, 1.05])
    plt.xlabel('False Positive Rate')
    plt.ylabel('True Positive Rate')
    plt.title('ROC Curve')
    plt.legend(loc="lower right")
    plt.grid(True, alpha=0.3)
    plt.savefig('roc_auc_curve.png', dpi=300, bbox_inches='tight')
    plt.show()