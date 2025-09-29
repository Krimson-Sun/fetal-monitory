package utils

import (
	"math/rand"
	"time"
)

// RandomInt возвращает случайное число в диапазоне [min, max]
func RandomInt(min, max int) int {
	if min >= max {
		return min
	}
	return rand.Intn(max-min+1) + min
}

// RandomDuration возвращает случайную длительность в диапазоне [min, max]
func RandomDuration(min, max time.Duration) time.Duration {
	if min >= max {
		return min
	}
	return min + time.Duration(rand.Int63n(int64(max-min)))
}

// AddJitter добавляет случайное отклонение к базовому времени
func AddJitter(baseTime time.Time, jitter time.Duration) time.Time {
	if jitter == 0 {
		return baseTime
	}
	jitterAmount := time.Duration(float64(jitter) * (rand.Float64()*2 - 1))
	return baseTime.Add(jitterAmount)
}

// LinearInterpolation линейная интерполяция между двумя значениями
func LinearInterpolation(start, end int, progress float64) int {
	if progress <= 0 {
		return start
	}
	if progress >= 1 {
		return end
	}
	return start + int(float64(end-start)*progress)
}

// TimestampToUnixMillis конвертирует время в Unix миллисекунды
func TimestampToUnixMillis(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

// UnixMillisToTimestamp конвертирует Unix миллисекунды во время
func UnixMillisToTimestamp(millis int64) time.Time {
	return time.Unix(0, millis*int64(time.Millisecond))
}

// CalculateAverage вычисляет среднее значение массива чисел
func CalculateAverage(values []int) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0
	for _, v := range values {
		sum += v
	}
	return float64(sum) / float64(len(values))
}

// Min возвращает минимальное из двух чисел
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max возвращает максимальное из двух чисел
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Clamp ограничивает значение в диапазоне [min, max]
func Clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
