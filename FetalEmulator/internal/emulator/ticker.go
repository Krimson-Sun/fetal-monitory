package emulator

import (
	"context"
	"math/rand"
	"time"
)

// Ticker управляет временными интервалами эмулятора
type Ticker struct {
	interval time.Duration
	jitter   time.Duration // Случайное отклонение для реалистичности
}

func NewTicker(interval, jitter time.Duration) *Ticker {
	return &Ticker{
		interval: interval,
		jitter:   jitter,
	}
}

// Tick возвращает канал, который отправляет метки времени с заданным интервалом
func (t *Ticker) Tick(ctx context.Context) <-chan time.Time {
	tickChan := make(chan time.Time)

	go func() {
		defer close(tickChan)

		// Первый тик сразу
		tickChan <- time.Now()

		ticker := time.NewTicker(t.interval)
		defer ticker.Stop()

		for {
			select {
			case tickTime := <-ticker.C:
				// Добавляем случайное отклонение для реалистичности
				if t.jitter > 0 {
					jitteredTime := t.addJitter(tickTime)
					select {
					case tickChan <- jitteredTime:
					case <-ctx.Done():
						return
					}
				} else {
					select {
					case tickChan <- tickTime:
					case <-ctx.Done():
						return
					}
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return tickChan
}

func (t *Ticker) addJitter(baseTime time.Time) time.Time {
	// Реализация в отдельном файле utils
	jitterDuration := time.Duration(float64(t.jitter) * (rand.Float64()*2 - 1))
	return baseTime.Add(jitterDuration)
}

// OneShotTick создает одноразовый тик через заданное время
func (t *Ticker) OneShotTick(ctx context.Context, delay time.Duration) <-chan time.Time {
	tickChan := make(chan time.Time, 1)

	go func() {
		select {
		case <-time.After(delay):
			tickChan <- time.Now()
		case <-ctx.Done():
		}
		close(tickChan)
	}()

	return tickChan
}
