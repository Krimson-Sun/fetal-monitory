-- Таблица сессий
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(64) PRIMARY KEY,
    status VARCHAR(20) NOT NULL,
    started_at TIMESTAMP NOT NULL,
    stopped_at TIMESTAMP,
    saved_at TIMESTAMP,
    total_duration_ms BIGINT DEFAULT 0,
    total_data_points BIGINT DEFAULT 0,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_sessions_status ON sessions(status);
CREATE INDEX idx_sessions_started_at ON sessions(started_at DESC);

-- Таблица агрегированных метрик
CREATE TABLE IF NOT EXISTS session_metrics (
    session_id VARCHAR(64) PRIMARY KEY REFERENCES sessions(id) ON DELETE CASCADE,
    stv DOUBLE PRECISION DEFAULT 0,
    ltv DOUBLE PRECISION DEFAULT 0,
    baseline_heart_rate DOUBLE PRECISION DEFAULT 0,
    total_accelerations INTEGER DEFAULT 0,
    total_decelerations INTEGER DEFAULT 0,
    late_decelerations INTEGER DEFAULT 0,
    late_deceleration_ratio DOUBLE PRECISION DEFAULT 0,
    total_contractions INTEGER DEFAULT 0,
    accel_decel_ratio DOUBLE PRECISION DEFAULT 0,
    stv_trend DOUBLE PRECISION DEFAULT 0,
    bpm_trend DOUBLE PRECISION DEFAULT 0,
    data_points INTEGER DEFAULT 0,
    time_span_sec DOUBLE PRECISION DEFAULT 0,
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Таблица событий (акселерации, децелерации, сокращения)
CREATE TABLE IF NOT EXISTS session_events (
    id BIGSERIAL PRIMARY KEY,
    session_id VARCHAR(64) NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    event_type VARCHAR(20) NOT NULL, -- 'acceleration', 'deceleration', 'contraction'
    start_time DOUBLE PRECISION NOT NULL,
    end_time DOUBLE PRECISION NOT NULL,
    duration DOUBLE PRECISION NOT NULL,
    amplitude DOUBLE PRECISION NOT NULL,
    is_late BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_session_events_session_id ON session_events(session_id);
CREATE INDEX idx_session_events_type ON session_events(event_type);
CREATE INDEX idx_session_events_start_time ON session_events(start_time);

-- Таблица временных рядов (STV, LTV по окнам)
CREATE TABLE IF NOT EXISTS session_timeseries (
    id BIGSERIAL PRIMARY KEY,
    session_id VARCHAR(64) NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    metric_type VARCHAR(20) NOT NULL, -- 'stv', 'ltv'
    time_index INTEGER NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    window_duration DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_session_timeseries_session_id ON session_timeseries(session_id);
CREATE INDEX idx_session_timeseries_type ON session_timeseries(metric_type);
CREATE INDEX idx_session_timeseries_time_index ON session_timeseries(time_index);

-- Таблица raw данных (опционально, для полного replay)
CREATE TABLE IF NOT EXISTS session_raw_data (
    id BIGSERIAL PRIMARY KEY,
    session_id VARCHAR(64) NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    batch_ts_ms BIGINT NOT NULL,
    metric_type VARCHAR(10) NOT NULL, -- 'FHR', 'UC'
    data JSONB NOT NULL, -- массив {time_sec, value}
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_session_raw_data_session_id ON session_raw_data(session_id);
CREATE INDEX idx_session_raw_data_metric_type ON session_raw_data(metric_type);
CREATE INDEX idx_session_raw_data_batch_ts ON session_raw_data(batch_ts_ms);

-- Комментарии к таблицам
COMMENT ON TABLE sessions IS 'Основная таблица мониторинговых сессий';
COMMENT ON TABLE session_metrics IS 'Агрегированные медицинские метрики сессии';
COMMENT ON TABLE session_events IS 'События: акселерации, децелерации и маточные сокращения';
COMMENT ON TABLE session_timeseries IS 'Временные ряды метрик (STV, LTV)';
COMMENT ON TABLE session_raw_data IS 'Raw данные для replay и анализа';

-- Функция для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Триггер для session_metrics
CREATE TRIGGER update_session_metrics_updated_at 
    BEFORE UPDATE ON session_metrics
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

