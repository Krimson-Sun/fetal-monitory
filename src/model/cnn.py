import pandas as pd
import numpy as np
from sklearn.model_selection import train_test_split
from sklearn.utils.class_weight import compute_class_weight
from sklearn.metrics import classification_report, roc_auc_score
import torch
import torch.nn as nn
import torch.nn.functional as F
from torch.utils.data import Dataset, DataLoader


class SignalDataset(Dataset):
    def __init__(self, data, labels):
        self.data = data
        self.labels = labels

    def __len__(self):
        return len(self.data)

    def __getitem__(self, idx):
        bpm, uterus = self.data[idx]
        # Преобразуем в numpy array и убеждаемся, что оба сигнала одинаковой длины
        bpm = np.array(bpm, dtype=np.float32)
        uterus = np.array(uterus, dtype=np.float32)

        # Проверяем длины и делаем паддинг если нужно
        if len(bpm) != len(uterus):
            max_len = max(len(bpm), len(uterus))
            bpm = np.pad(bpm, (0, max_len - len(bpm)), mode="constant")
            uterus = np.pad(uterus, (0, max_len - len(uterus)), mode="constant")

        x = torch.tensor(np.stack([bpm, uterus]), dtype=torch.float32)  # [2, T]
        y = torch.tensor(self.labels[idx], dtype=torch.long)
        return x, y


def smart_collate_fn(batch):
    xs, ys = zip(*batch)

    # Находим максимальную длину в КОНКРЕТНОМ батче
    max_len = max(x.shape[1] for x in xs)

    padded = []
    for x in xs:
        pad_len = max_len - x.shape[1]
        if pad_len > 0:
            # Паддим нулями ТОЛЬКО если нужно
            x = F.pad(x, (0, pad_len))
        padded.append(x)

    X = torch.stack(padded)
    Y = torch.tensor(ys)
    return X, Y


class ImprovedCNN(nn.Module):
    def __init__(self, n_classes=2, dropout_rate=0.3):
        super().__init__()

        self.conv1 = nn.Conv1d(2, 64, kernel_size=7, padding=3)
        self.bn1 = nn.BatchNorm1d(64)

        self.conv2 = nn.Conv1d(64, 128, kernel_size=5, padding=2)
        self.bn2 = nn.BatchNorm1d(128)

        self.conv3 = nn.Conv1d(128, 256, kernel_size=3, padding=1)
        self.bn3 = nn.BatchNorm1d(256)

        self.conv4 = nn.Conv1d(256, 512, kernel_size=3, padding=1)
        self.bn4 = nn.BatchNorm1d(512)

        self.pool = nn.AdaptiveAvgPool1d(1)
        self.dropout = nn.Dropout(dropout_rate)

        self.classifier = nn.Sequential(
            nn.Linear(512, 128),
            nn.ReLU(),
            nn.Dropout(dropout_rate),
            nn.Linear(128, 32),
            nn.ReLU(),
            nn.Dropout(dropout_rate),
            nn.Linear(32, n_classes),
        )

    def forward(self, x):
        x = F.relu(self.bn1(self.conv1(x)))
        x = self.dropout(x)

        x = F.relu(self.bn2(self.conv2(x)))
        x = self.dropout(x)

        x = F.relu(self.bn3(self.conv3(x)))
        x = self.dropout(x)

        x = F.relu(self.bn4(self.conv4(x)))
        x = self.pool(x).squeeze(-1)

        x = self.classifier(x)
        return x


def train_epoch(model, train_loader, criterion, optimizer, device):
    model.train()
    total_loss = 0.0
    all_preds = []
    all_targets = []

    for X, y in train_loader:
        X, y = X.to(device), y.to(device)
        optimizer.zero_grad()
        out = model(X)
        loss = criterion(out, y)
        loss.backward()

        torch.nn.utils.clip_grad_norm_(model.parameters(), max_norm=1.0)
        optimizer.step()

        total_loss += loss.item() * X.size(0)
        all_preds.extend(out.argmax(dim=1).cpu().numpy())
        all_targets.extend(y.cpu().numpy())

    avg_loss = total_loss / len(train_loader.dataset)
    accuracy = np.mean(np.array(all_preds) == np.array(all_targets))

    return avg_loss, accuracy


def evaluate_model(model, test_loader, device):
    model.eval()
    all_preds = []
    all_probs = []
    all_targets = []

    with torch.no_grad():
        for X, y in test_loader:
            X, y = X.to(device), y.to(device)
            out = model(X)
            probs = F.softmax(out, dim=1)

            all_preds.extend(out.argmax(dim=1).cpu().numpy())
            all_probs.extend(probs.cpu().numpy())
            all_targets.extend(y.cpu().numpy())

    return np.array(all_preds), np.array(all_probs), np.array(all_targets)


if __name__ == "__main__":
    # Загрузка и подготовка данных
    data = []
    labels = []

    filtered_df = pd.read_pickle(
        "/home/be2r/hackathons/fetal-monitory/large_data/preprocessed/filtered_signals_df.pkl"
    )

    # Сначала соберем все длины чтобы посмотреть статистику
    bpm_lengths = []
    uterus_lengths = []

    for pid, df_patient in filtered_df.groupby("patient_uid"):
        bpm = (
            df_patient[df_patient["sensor_type"] == "bpm"]
            .sort_values("time_sec")["value"]
            .to_numpy()
        )
        uterus = (
            df_patient[df_patient["sensor_type"] == "uterus"]
            .sort_values("time_sec")["value"]
            .to_numpy()
        )

        # Сохраняем исходные сигналы БЕЗ обрезки
        data.append((bpm, uterus))
        bpm_lengths.append(len(bpm))
        uterus_lengths.append(len(uterus))

        target = 1 if pid.split("_")[0] == "hypoxia" else 0
        labels.append(target)

    labels = np.array(labels)  # Преобразуем в numpy array

    print(f"Всего пациентов: {len(data)}")
    print(f"Распределение классов: {np.bincount(labels)}")
    print(
        f"Длины BPM сигналов: min={min(bpm_lengths)}, max={max(bpm_lengths)}, mean={np.mean(bpm_lengths):.1f}"
    )
    print(
        f"Длины uterus сигналов: min={min(uterus_lengths)}, max={max(uterus_lengths)}, mean={np.mean(uterus_lengths):.1f}"
    )

    # Вычисление весов классов
    class_weights = compute_class_weight("balanced", classes=np.array([0, 1]), y=labels)
    class_weights = torch.tensor(class_weights, dtype=torch.float32)
    print(f"Веса классов: {class_weights}")

    # Стратифицированное разделение
    train_data, test_data, train_labels, test_labels = train_test_split(
        data, labels, test_size=0.2, stratify=labels, random_state=42
    )

    # Создание датасетов и загрузчиков
    train_dataset = SignalDataset(train_data, train_labels)
    test_dataset = SignalDataset(test_data, test_labels)

    # Используем smart_collate_fn для паддинга
    train_loader = DataLoader(
        train_dataset, batch_size=4, shuffle=True, collate_fn=smart_collate_fn
    )  # Уменьшил batch_size
    test_loader = DataLoader(
        test_dataset, batch_size=4, shuffle=False, collate_fn=smart_collate_fn
    )

    # Инициализация модели и оптимизатора
    device = torch.device("cpu")
    print(f"Используемое устройство: {device}")

    model = ImprovedCNN(n_classes=2, dropout_rate=0.3).to(device)
    criterion = nn.CrossEntropyLoss(weight=class_weights.to(device))
    optimizer = torch.optim.AdamW(model.parameters(), lr=1e-4, weight_decay=1e-4)

    # Убрал verbose параметр
    scheduler = torch.optim.lr_scheduler.ReduceLROnPlateau(
        optimizer, mode="min", patience=3, factor=0.5
    )

    # Обучение
    best_acc = 0
    best_auc = 0
    for epoch in range(30):  # Уменьшил количество эпох
        train_loss, train_acc = train_epoch(
            model, train_loader, criterion, optimizer, device
        )

        # Валидация
        val_preds, val_probs, val_targets = evaluate_model(model, test_loader, device)
        val_acc = np.mean(val_preds == val_targets)
        val_auc = roc_auc_score(val_targets, val_probs[:, 1])

        scheduler.step(train_loss)

        print(f"Epoch {epoch+1:02d}:")
        print(f"  Train Loss: {train_loss:.4f}, Train Acc: {train_acc:.3f}")
        print(f"  Val Acc: {val_acc:.3f}, Val AUC: {val_auc:.3f}")

        # Сохранение лучшей модели по AUC
        if val_auc > best_auc:
            best_auc = val_auc
            best_acc = val_acc
            torch.save(model.state_dict(), "best_model.pth")
            print(f"  -> Новый лучший AUC! Модель сохранена.")

        # Early stopping если loss NaN или слишком высокий
        if np.isnan(train_loss) or (epoch > 10 and train_loss > 2.0):
            print("  -> Ранняя остановка из-за проблем с обучением")
            break

    # Финальная оценка
    model.load_state_dict(torch.load("best_model.pth"))
    test_preds, test_probs, test_targets = evaluate_model(model, test_loader, device)

    print("\n" + "=" * 50)
    print("ФИНАЛЬНЫЕ РЕЗУЛЬТАТЫ:")
    print("=" * 50)
    print(f"Точность: {np.mean(test_preds == test_targets):.3f}")
    print(f"AUC-ROC: {roc_auc_score(test_targets, test_probs[:, 1]):.3f}")
    print("\nClassification Report:")
    print(
        classification_report(
            test_targets, test_preds, target_names=["Normal", "Hypoxia"]
        )
    )

    # Вывод информации о паддинге
    print(f"\nИнформация о паддинге:")
    print(f"Максимальная длина BPM в данных: {max(bpm_lengths)}")
    print(f"Максимальная длина uterus в данных: {max(uterus_lengths)}")
