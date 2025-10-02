export const METRIC_LIMITS = {
    'stv': (value)=> value>4?'green':value>3?'yellow':'red',
    'ltv': (value)=> value>40?'green':value>30?'yellow':'red',
    'baseline': (value)=> value>160?'yellow':value<110?'yellow':'green',
    'late-decel': (value)=> value>50?'red':value>10?'yellow':'green',
    'mean-contractions-amplitude': (value)=> value>20?'green':value>10?'yellow':'red',
    'accelerations-rate': (value)=> value>2? 'green':value>1?'yellow':'red',

    'prediction': (value) => value>0.8? 'red alert-pulse':value>0.2?'yellow':'green'
}