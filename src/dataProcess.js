import { Chart } from "chart.js/auto";
import zoomPlugin from 'chartjs-plugin-zoom';
import { METRIC_LIMITS } from "./metricNorm";
import { EXAMPLE_DATA } from "./exampleData";
import { setSaveDelBtns } from "./recording";

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

let time = 29.5
let dataOnlineExample = JSON.parse(JSON.stringify(EXAMPLE_DATA))
export function updateData() {
    let data = dataOnlineExample
    const appendDots = Math.round((Math.random()*3+1))

    for (let i=0; i<appendDots; i++){
      data['records']['filtered_bpm_batch']['time_sec'].push(time)
      data['records']['filtered_bpm_batch']['value'].push(60+Math.random()*120)
      data['records']['filtered_uterus_batch']['time_sec'].push(time)
      data['records']['filtered_uterus_batch']['value'].push(Math.random()*100)
      time += Math.random()
    }

    if (data['records']['filtered_bpm_batch']['time_sec'].length>52)
      data['records']['filtered_bpm_batch']['time_sec'] = data['records']['filtered_bpm_batch']['time_sec'].slice(-52)
    if (data['records']['filtered_bpm_batch']['value'].length>52)
      data['records']['filtered_bpm_batch']['value'] = data['records']['filtered_bpm_batch']['value'].slice(-52)
    if (data['records']['filtered_uterus_batch']['time_sec'].length>52)
      data['records']['filtered_uterus_batch']['time_sec'] = data['records']['filtered_uterus_batch']['time_sec'].slice(-52)
    if (data['records']['filtered_uterus_batch']['value'].length>52)
      data['records']['filtered_uterus_batch']['value'] = data['records']['filtered_uterus_batch']['value'].slice(-52)

    updateOnlineData(data['records'], data['prediction']);
}

export function updateOnlineData(data) {
    let j;
    let min_bpm = hrChart.options.scales.y.min
    let max_bpm = hrChart.options.scales.y.max
    let min_uc = uterineChart.options.scales.y.min
    let max_uc = uterineChart.options.scales.y.max


    const bpmFiltered = data['filtered_bpm_batch']
    j = Math.max(0, hrChart.data.datasets[0].data.length - 51)
    while (hrChart.data.datasets[0].data[j] && hrChart.data.datasets[0].data[j].x < bpmFiltered['time_sec'][0] * 1000) j++;
    for (let k=0; k<bpmFiltered['time_sec'].length; k++){
      if (k+j < hrChart.data.datasets[0].data.length)
        hrChart.data.datasets[0].data[k+j] = {
          x:bpmFiltered['time_sec'][k] * 1000,
          y:bpmFiltered['value'][k]
        }
      else hrChart.data.datasets[0].data.push(
        {
          x:bpmFiltered['time_sec'][k] * 1000,
          y:bpmFiltered['value'][k]
        }
      )
      if (bpmFiltered['value'][k]>max_bpm) max_bpm = bpmFiltered['value'][k]
      if (bpmFiltered['value'][k]<min_bpm) min_bpm = bpmFiltered['value'][k]
    }

    const ucFiltered = data['filtered_uterus_batch']
    j = Math.max(0, uterineChart.data.datasets[0].data.length - 51)
    while (uterineChart.data.datasets[0].data[j] && uterineChart.data.datasets[0].data[j].x < ucFiltered['time_sec'][0] * 1000) j++;
    for (let i=0; i<ucFiltered['time_sec'].length; i++){
      if (i+j < uterineChart.data.datasets[0].data.length)
        uterineChart.data.datasets[0].data[i+j] = {
          x:ucFiltered['time_sec'][i] * 1000,
          y:ucFiltered['value'][i]
        }
      else uterineChart.data.datasets[0].data.push(
        {
          x:ucFiltered['time_sec'][i] * 1000,
          y:ucFiltered['value'][i]
        }
      )
      if (ucFiltered['value'][i]>max_uc) max_uc = ucFiltered['value'][i]
      if (ucFiltered['value'][i]<min_uc) min_uc = ucFiltered['value'][i]
      if (ucFiltered['value'][i] > 100) console.log('!!!', ucFiltered['value'][i])
    }

    hrChart.data.datasets[1].data = []
    hrChart.data.datasets[2].data = []
    uterineChart.data.datasets[1].data = []
    
    let meanContAmplitude = setAccelDecelContr(data['accelerations'], data['decelerations'], data['contractions'])

    let min_time;
    let max_time;
    try{
      min_time = Math.min(hrChart.data.datasets[0].data[0].x,
                      uterineChart.data.datasets[0].data[0].x);
      max_time = Math.max(hrChart.data.datasets[0].data[hrChart.data.datasets[0].data.length-1].x,
                      uterineChart.data.datasets[0].data[hrChart.data.datasets[0].data.length-1].x);
    } catch(error){
      min_time = 0;
      max_time = 30
    }
    
    const stv = data['stv'];
    const ltv = data['ltv'];
    const lateDecel = data['late_deceleration_ratio']*100;
    const baseline = data['baseline_heart_rate'];

    hrChart.data.datasets[3].data = [
      {x:hrChart.data.datasets[0].data[0].x, y:baseline},
      {x:hrChart.data.datasets[0].data[hrChart.data.datasets[0].data.length-1].x, y: baseline}
    ]

    updateMetric('baseline', baseline, Math.round)
    updateMetric('stv', stv, (value)=>value == 0? null:value.toFixed(1))
    updateMetric('ltv', ltv, (value)=>value == 0? null:value.toFixed(1))
    updateMetric('late-decel', lateDecel, (value)=> data['total_decelerations'] == 0? null:`${value.toFixed(1)}%`)
    updateMetric('accelerations-rate', (data['total_accelerations']*60000 / (max_time-min_time)).toFixed(0))
    updateMetric('mean-contractions-amplitude', meanContAmplitude, (value)=>value.toFixed(0))
    
    document.getElementById('accelerations-value').textContent = data['total_accelerations'];
    document.getElementById('decelerations-value').textContent = data['total_decelerations'];
    document.getElementById('contractions-value').textContent = data['total_contractions'];

    document.getElementById('forecast-value').textContent = `${(data['prediction']*100).toFixed(0)}%`;
    const predictionStatus = METRIC_LIMITS['prediction'](data['prediction'])
    document.getElementById('forecast-value').className = `forecast-value forecast-${predictionStatus}`;
    document.getElementById('status-badge').className = `status-badge status-${predictionStatus}`;
    document.getElementById('status-badge').textContent = 
        predictionStatus == 'green'?'Все в порядке':predictionStatus == 'yellow'? 'Требуется внимание':'Риск осложнений';

    // Обновление всех графиков
    hrChart.update();
    uterineChart.update();
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


export function setDataToCharts(data, prediction){
  resetCharts()
  setSaveDelBtns()

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

  let meanContAmplitude = setAccelDecelContr(d['accelerations'], d['decelerations'], d['contractions']);

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

function setAccelDecelContr(accelerations, decelerations, contractions){
  let meanContAmplitude = 0;

  for (let i=0; i<contractions.length; i++){
    let contraction = contractions[i]
    uterineChart.data.datasets[1].data.push({
      x: uterineChart.data.datasets[0].data[contraction['start']].x,
      y: uterineChart.data.datasets[0].data[contraction['start']].y,
    })
    meanContAmplitude += contraction['amplitude']/contractions.length;
  }

  for (let i=0; i<accelerations.length; i++){
    let accel = accelerations[i]
    hrChart.data.datasets[1].data.push({
      x: hrChart.data.datasets[0].data[accel['start']].x,
      y: hrChart.data.datasets[0].data[accel['start']].y,
    })
  }

  for (let i=0; i<decelerations.length; i++){
    let accel = decelerations[i]
    hrChart.data.datasets[2].data.push({
      x: hrChart.data.datasets[0].data[accel['start']].x,
      y: hrChart.data.datasets[0].data[accel['start']].y,
    })
  }

  return meanContAmplitude
}