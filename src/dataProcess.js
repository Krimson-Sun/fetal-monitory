import { Chart } from "chart.js/auto";
import zoomPlugin from 'chartjs-plugin-zoom';
import { METRIC_LIMITS } from "./metricNorm";

const METRIC_LIST = ['stv', 'ltv', 'baseline_heart_rate', 'total_decelerations', 'late_decelerations',
  'late_deceleration_ratio', 'total_accelerations', 'accel_decel_ratio', 'total_contractions', 
]

// Регистрируем плагин
Chart.register(zoomPlugin);

let hrChart;
let uterineChart;

let fa = false

export function resetCharts(){
      if (hrChart) hrChart.destroy();
      if (uterineChart) uterineChart.destroy();
      
      const hrCanvas = document.getElementById('hrChart').getContext('2d');
      hrChart = new Chart(hrCanvas, {
        type: 'line',
        data: {
          datasets: [
            {
              label: 'ЧСС плода',
              data: [],
              borderColor: '#0096A6',
              borderWidth: 2.5,
              pointRadius: 0,
              yAxisID: 'y',
            },
            {
              label: 'Ускорения',
              data: [],
              borderColor: '#4CAF50',
              borderWidth: 2,
              pointRadius: 5,
              pointBackgroundColor: '#4CAF50',
              pointBorderWidth: 1,
              pointStyle: 'triangle',
              yAxisID: 'y',
              showLine: false
            },
            {
              label: 'Децелерации',
              data: [],
              borderColor: '#F44336',
              borderWidth: 2,
              pointRadius: 5,
              pointBackgroundColor: '#F44336',
              pointBorderWidth: 1,
              pointStyle: 'triangle',
              yAxisID: 'y',
              showLine: false
            },
            {
              label: 'БЧСС',
              data: [],
              borderColor: '#616161ff',
              borderWidth: 2.5,
              pointRadius: 0,
              yAxisID: 'y',
              borderDash: [6, 6]
            },
          ]
        },
        options: {
          responsive: true,
          //maintainAspectRatio: false,
          elements: {
            line: {
              tension: 0.3
            }
          },
          plugins: {
            legend: {
              display: false
            },
            tooltip: {
              enabled: false
            },
            zoom: {
                pan: {
                  onPan({ chart }) {
                    uterineChart.zoomScale(
                      'x',
                      { min: Math.trunc(chart.scales.x.min), max: Math.trunc(chart.scales.x.max) },
                      'none'
                    );
                  },
                  enabled: true,
                  mode: 'x',
                },
                zoom: {
                  mode: 'x',
                  onZoom: ({ chart }) => {
                    const xScale = chart.options.scales.x;
                    uterineChart.options.scales.x.min = xScale.min;
                    uterineChart.options.scales.x.max = xScale.max;
                    uterineChart.update();
                  },
                  drag: {
                    enabled: false
                  },
                  pinch: {
                    enabled: true
                  },
                  wheel: {
                    enabled: true
                  }
                }
              }
          },
          scales: {
            x: {
              type: 'linear',
              position: 'bottom',
              grid: {
                display: false
              },
              ticks: {
                color: '#666',
                font: {
                  size: 12
                },
                callback: function(value) {
                  return `${Math.floor(value / 60000)}:${String(Math.floor(value / 1000) % 60).padStart(2,'0')}`;
                }
              }
            },
            y: {
              type: 'linear',
              position: 'left',
              min: 50,
              max: 200,
              grid: {
                color: 'rgba(200, 200, 200, 0.3)',
                lineWidth: 1,
                drawBorder: false
              },
              ticks: {
                color: '#666',
                font: {
                  size: 12
                }
              },
              title: {
                display: true,
                text: 'ЧСС плода (уд/мин)',
                color: '#666',
                font: {
                  size: 13
                }
              }
            }
          }
        }
      });
      
      // Инициализация графика маточных сокращений
      const uterineCanvas = document.getElementById('uterineChart').getContext('2d');
      uterineChart = new Chart(uterineCanvas, {
        type: 'line',
        data: {
          datasets: [
            {
              label: 'Маточные сокращения',
              data: [],
              borderColor: '#0096A6',
              borderWidth: 2.5,
              pointRadius: 0,
              yAxisID: 'y'
            },
            {
              label: 'Схватки',
              data: [],
              borderColor: '#4CAF50',
              borderWidth: 2,
              pointRadius: 5,
              pointBackgroundColor: '#4CAF50',
              pointBorderWidth: 1,
              pointStyle: 'triangle',
              yAxisID: 'y',
              showLine: false
            },
          ]
        },
        options: {
          responsive: true,
          //maintainAspectRatio: false,
          elements: {
            line: {
              tension: 0.3
            }
          },
          plugins: {
            legend: {
              display: false
            },
            tooltip: {
              enabled: false
            },
            zoom:{
              pan: {
                onPan({ chart }) {
                  hrChart.zoomScale(
                    'x',
                    { min: Math.trunc(chart.scales.x.min), max: Math.trunc(chart.scales.x.max) },
                    'none'
                  );
                },
                enabled: true,
                mode: 'x'
              },
              zoom: {
                wheel: { enabled: true },
                pinch: { enabled: true },
                mode: 'x',
                onZoom: ({ chart }) => {
                  const xScale = chart.options.scales.x;
                  hrChart.options.scales.x.min = xScale.min;
                  hrChart.options.scales.x.max = xScale.max;
                  hrChart.update();
                }
              }
            }
          },
          scales: {
            x: {
              type: 'linear',
              position: 'bottom',
              grid: {
                display: false
              },
              ticks: {
                color: '#666',
                font: {
                  size: 12
                },
                callback: function(value) {
                  return `${Math.floor(value / 60000)}:${String(Math.floor(value / 1000) % 60).padStart(2,'0')}`;
                }
              }
            },
            y: {
              type: 'linear',
              position: 'left',
              min: 0,
              max: 100,
              grid: {
                color: 'rgba(200, 200, 200, 0.3)',
                lineWidth: 1,
                drawBorder: false
              },
              ticks: {
                color: '#666',
                font: {
                  size: 12
                }
              },
              title: {
                display: true,
                text: 'Маточные сокращения (%)',
                color: '#666',
                font: {
                  size: 13
                }
              }
            }
          }
        }
      });

      updateMetric('baseline', 0, ()=>null)
      updateMetric('stv', 0, ()=>null)
      updateMetric('ltv', 0, ()=>null)
      updateMetric('late-decel', 0, ()=>null)
      updateMetric('accelerations-rate', 0, ()=>null)
      updateMetric('mean-contractions-amplitude', 0, ()=>null)
      
      document.getElementById('accelerations-value').textContent = '-';
      document.getElementById('decelerations-value').textContent = '-';
      document.getElementById('contractions-value').textContent = '-';

      document.getElementById('forecast-value').textContent = '';
      document.getElementById('forecast-value').className = `forecast-value forecast-gray`;
      document.getElementById('status-badge').className = '';
      document.getElementById('status-badge').textContent = ''
      
}

export function updateData(initDate) {
    const now = Date.now() - initDate;

    // Обновление ЧСС
    const hrValue = 130 + Math.random() * 20;
    hrChart.data.datasets[0].data.push({ x: now, y: hrValue });

    // Симуляция ускорений (зеленые треугольники)
    if (Math.random() < 0.05) {
        hrChart.data.datasets[1].data.push({ 
        x: now, 
        y: hrValue + 15,
        label: 'Ускорение'
        });
    }

    // Симуляция децелераций (красные треугольники)
    if (Math.random() < 0.03) {
        const decelType = Math.random() < 0.5 ? 'R' : 'L';
        hrChart.data.datasets[2].data.push({ 
        x: now, 
        y: hrValue - 25,
        label: `Децелерация ${decelType}`
        });
    }

    // Ограничение данных для ЧСС
    const maxPoints = 600;
    if (hrChart.data.datasets[0].data.length > maxPoints) {
        hrChart.data.datasets[0].data = hrChart.data.datasets[0].data.slice(-maxPoints);
        hrChart.data.datasets[1].data = hrChart.data.datasets[1].data.slice(-maxPoints);
        hrChart.data.datasets[2].data = hrChart.data.datasets[2].data.slice(-maxPoints);
    }

    hrChart.options.scales.x.min = hrChart.data.datasets[0].data[0].x;
    hrChart.options.scales.x.max = now;

    // Обновление маточных сокращений
    const uterineValue = Math.random() * 50;
    uterineChart.data.datasets[0].data.push({ x: now, y: uterineValue });

    // Ограничение данных для маточных сокращений
    if (uterineChart.data.datasets[0].data.length > maxPoints) {
        uterineChart.data.datasets[0].data = uterineChart.data.datasets[0].data.slice(-maxPoints);
    }
    uterineChart.options.scales.x.min = uterineChart.data.datasets[0].data[0].x;
    uterineChart.options.scales.x.max = now;

    // Обновление метрик
    const stv = 7.2 + (Math.random() - 0.5) * 2;
    const ltv = 12.5 + (Math.random() - 0.5) * 3;
    const lateDecel = Math.random() * 10;
    const accelerations = Math.floor(Math.random() * 5);

    document.getElementById('stv-value').textContent = stv.toFixed(1);
    document.getElementById('ltv-value').textContent = ltv.toFixed(1);
    document.getElementById('late-decel-value').textContent = `${lateDecel.toFixed(1)}%`;
    document.getElementById('accelerations-value').textContent = accelerations;

    // Обновление статуса метрик
    // updateMetricStatus('stv', stv);
    // updateMetricStatus('ltv', ltv);
    // updateMetricStatus('late-decel', lateDecel);
    // updateMetricStatus('accelerations', accelerations);

    // Обновление прогноза
    const forecast = Math.random() * 20;
    document.getElementById('forecast-value').textContent = `${forecast.toFixed(0)}%`;

    // Обновление цвета прогноза
    if (forecast < 5) {
        document.getElementById('forecast-value').className = 'forecast-value forecast-green';
        document.getElementById('status-badge').className = 'status-badge status-green';
        document.getElementById('status-badge').textContent = 'Все в порядке';
    } else if (forecast < 15) {
        document.getElementById('forecast-value').className = 'forecast-value forecast-yellow';
        document.getElementById('status-badge').className = 'status-badge status-yellow';
        document.getElementById('status-badge').textContent = 'Повышенный риск';
    } else {
        document.getElementById('forecast-value').className = 'forecast-value forecast-red alert-pulse';
        document.getElementById('status-badge').className = 'status-badge status-red alert-pulse';
        document.getElementById('status-badge').textContent = 'Экстренное вмешательство';
    }

    // Обновление всех графиков
    hrChart.update('none');
    uterineChart.update('none');
    //trendChart.update('none');
}

    // Обновление статуса метрик
function updateMetric(id, value, parser=null) {
    const valueField = document.getElementById(`${id}-value`);
    if (!valueField) return;
    const parsed = parser?parser(value):value
    valueField.textContent = parsed? parsed:'-';
    const element = document.getElementById(`${id}-status`);
    if (!element) return;
    element.className = `metric-status metric-status-${parsed?METRIC_LIMITS[id](value):'gray'}`;
}


export function setDataToCharts(data){
  resetCharts()

  const d = data['records']
  let min_bpm = 1e10
  let max_bpm = -1e10
  let min_uc = 1e10
  let max_uc = -1e10

  for (let i=0; i<Math.min(d['filtered_bpm_batch']['time_sec'].length, d['filtered_bpm_batch']['value'].length); i++){
    hrChart.data.datasets[0].data.push({x:d['filtered_bpm_batch']['time_sec'][i],y: d['filtered_bpm_batch']['value'][i]})
    if (d['filtered_bpm_batch']['value'][i]>max_bpm) max_bpm = d['filtered_bpm_batch']['value'][i]
    if (d['filtered_bpm_batch']['value'][i]<min_bpm) min_bpm = d['filtered_bpm_batch']['value'][i]
  }

  for (let i=0; i<Math.min(d['filtered_uterus_batch']['time_sec'].length, d['filtered_uterus_batch']['value'].length); i++){
    uterineChart.data.datasets[0].data.push({x:d['filtered_uterus_batch']['time_sec'][i],y: d['filtered_uterus_batch']['value'][i]})
    if (d['filtered_uterus_batch']['value'][i]>max_uc) max_uc = d['filtered_uterus_batch']['value'][i]
    if (d['filtered_uterus_batch']['value'][i]<min_uc) min_uc = d['filtered_uterus_batch']['value'][i]
  }

  let meanContAmplitude = 0;

  for (let i=0; i<d['contractions'].length; i++){
    let contraction = d['contractions'][i]
    uterineChart.data.datasets[1].data.push({
      x: uterineChart.data.datasets[0].data[contraction['start']].x,
      y: uterineChart.data.datasets[0].data[contraction['start']].y,
    })
    meanContAmplitude += contraction['amplitude']/d['contractions'].length;
  }

  for (let i=0; i<d['accelerations'].length; i++){
    let accel = d['accelerations'][i]
    hrChart.data.datasets[1].data.push({
      x: hrChart.data.datasets[0].data[accel['start']].x,
      y: hrChart.data.datasets[0].data[accel['start']].y,
    })
  }

  for (let i=0; i<d['decelerations'].length; i++){
    let accel = d['decelerations'][i]
    hrChart.data.datasets[2].data.push({
      x: hrChart.data.datasets[0].data[accel['start']].x,
      y: hrChart.data.datasets[0].data[accel['start']].y,
    })
  }

  const min_time = Math.min(hrChart.data.datasets[0].data[0].x,
                      uterineChart.data.datasets[0].data[0].x);
  const max_time = Math.max(hrChart.data.datasets[0].data[hrChart.data.datasets[0].data.length-1].x,
                      uterineChart.data.datasets[0].data[hrChart.data.datasets[0].data.length-1].x);

  hrChart.options.scales.x.min = min_time;
  hrChart.options.scales.x.max = max_time;

  hrChart.options.scales.y.min = min_bpm - 2;
  hrChart.options.scales.y.max = max_bpm + 2;
  uterineChart.options.scales.y.min = min_uc - 2;
  uterineChart.options.scales.y.max = max_uc + 2;

  // Обновление метрик
  const stv = d['stv'];
  const ltv = d['ltv'];
  const lateDecel = d['late_deceleration_ratio']*100;
  const baseline = d['baseline_heart_rate'];

  hrChart.data.datasets[3].data = [
    {x:hrChart.data.datasets[0].data[0].x, y:baseline},
    {x:hrChart.data.datasets[0].data[hrChart.data.datasets[0].data.length-1].x, y: baseline}
  ]

  updateMetric('baseline', baseline, Math.round)
  updateMetric('stv', stv, (value)=>value == 0? null:value.toFixed(1))
  updateMetric('ltv', ltv, (value)=>value == 0? null:value.toFixed(1))
  updateMetric('late-decel', lateDecel, (value)=> d['total_decelerations'] == 0? null:`${value.toFixed(1)}%`)
  updateMetric('accelerations-rate', (d['total_accelerations']*60000 / (max_time-min_time)).toFixed(0))
  updateMetric('mean-contractions-amplitude', meanContAmplitude, (value)=>value.toFixed(0))
  
  document.getElementById('accelerations-value').textContent = d['total_accelerations'];
  document.getElementById('decelerations-value').textContent = d['total_decelerations'];
  document.getElementById('contractions-value').textContent = d['total_contractions'];

  document.getElementById('forecast-value').textContent = `${(data['prediction']*100).toFixed(0)}%`;
  const predictionStatus = METRIC_LIMITS['prediction'](data['prediction'])
  document.getElementById('forecast-value').className = `forecast-value forecast-${predictionStatus}`;
  document.getElementById('status-badge').className = `status-badge status-${predictionStatus}`;
  document.getElementById('status-badge').textContent = 
      predictionStatus == 'green'?'Все в порядке':predictionStatus == 'yellow'? 'Требуется внимание':'Риск осложнений';

  uterineChart.update()
  hrChart.update()
}
