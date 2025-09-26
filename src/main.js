import Chart from 'chart.js/auto';
import { getRelativePosition } from 'chart.js/helpers';

const data1 = {
  labels: [],
  datasets: [
    {
      label: 'bpm',
      data: [],
      borderColor: '#0096A6', // Бирюзовый цвет
        borderWidth: 2.5,
        pointRadius: 5,
        pointBackgroundColor: '#ffffff',
        pointBorderColor: '#0096A6',
        pointBorderWidth: 2,
        pointHoverRadius: 6,
        fill: false,
        yAxisID: 'y',
        tension: 0.3 // Немного сглаживания
    }
  ]
};
 

// Создаем канвас элемент
const canvas1 = document.getElementById('canvas1');
const ctx1 = canvas1.getContext('2d');

const chart1 = new Chart(ctx1, {
  type: 'line',
  data: data1,
});

const data = {
  labels: [],
  datasets: [
    {
      label: 'uterus',
      data: [],
      borderColor: '#0096A6', // Бирюзовый цвет
        borderWidth: 2.5,
        pointRadius: 5,
        pointBackgroundColor: '#ffffff',
        pointBorderColor: '#0096A6',
        pointBorderWidth: 2,
        pointHoverRadius: 6,
        fill: false,
        yAxisID: 'y',
        tension: 0.3 // Немного сглаживания
    }
  ]
};
 

// Создаем канвас элемент
const canvas = document.getElementById('canvas2');
const ctx = canvas.getContext('2d');

let i=0;

const chart = new Chart(ctx, {
  type: 'line',
  data: data,
});

let interval = setInterval(()=>{
    chart.data.datasets[0].data.push(Math.random());
    chart.data.labels.push(i--)
    chart.update()

    chart1.data.datasets[0].data.push(Math.random());
    chart1.data.labels.push(i--)
    chart1.update()
}, 250)