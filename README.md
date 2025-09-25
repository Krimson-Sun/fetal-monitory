# Data analysis

В этой ветке представлен код для извлечения интерпретируемых медицинских признаков и фильтрации сигналов `bpm` и `uterus`. Для работы с данными необходимо поместить в директорию с блокнотом `data_analysis.ipynb` директорию `data` с данными:

```bash
├── 📁 data
│   ├── 📁 hypoxia
│   ├── 📁 regular
│   ├── 📄 hypoxia.xlsx
│   └── 📄 regular.xlsx
└── 📄 data_analysis.ipynb
```

Полученные в результате признаки могут быть интерпретируемы согласно информации в таблице [metric_interpretation.csv](metric_interpretation.csv) (требует доработки)