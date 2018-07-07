var first = true;
var chart = echarts.init(document.getElementById('chart-0'));
var Option = function() {
    return {
        title: {
            text: ''
        },
        tooltip: {
            trigger: 'axis',
            formatter: function(params) {
                params = params[0];
                var date = new Date(params.name);
                return date.getDate() + '/' + (date.getMonth() + 1) + '/' + date.getFullYear() + ' : ' + params.value[1];
            },
            axisPointer: {
                animation: false
            }
        },
        legend: {
            data: []
        },
        xAxis: {
            type: 'time',
            splitLine: {
                show: false
            }
        },
        yAxis: {
            type: 'value',
            axisLabel: {
                formatter: '{value} qps'
            },
            boundaryGap: [0, '100%'],
            splitLine: {
                show: false
            }
        },
        series: []
    }
}
var option = new Option();
var SeriesItem = function() {
    return {
        name: '',
        type: 'line',
        showSymbol: false,
        hoverAnimation: false,
        smooth: true,
        data: [],
        markPoint: {
            data: [{
                type: 'max',
                name: '最大值'
            },
            {
                type: 'min',
                name: '最小值'
            }]
        },
        markLine: {
            data: [{
                type: 'average',
                name: '平均值'
            }]
        },
    }
}
var ProcessStats = function() {
    return {
        id: 0,
        methods: [],
    }
}
var MethodStats = function() {
    return {
        name: '',
        qps: -1,
    }
}
function Convert(data) {
    var process = new ProcessStats();
    process.id = data.id;
    var method = new MethodStats();
    method.name = 'Total';
    method.qps = data.qps;
    process.methods.push(method);
    for (var name in data.method) {
        var method = new MethodStats();
        method.name = name;
        method.qps = data.method[name];
        process.methods.push(method);
    }
    return process
}
function GetStats() {
    jQuery.ajax({
        url: "http://##ListenAddr##/stats",
        type: 'get',
        async: 'true',
        dataType: 'json',
        success: function(data) {
            var process = Convert(data);
            if (!first) {
                if (process.methods.length != option.series.length) {
                    first = true
                } else if (('PID-'+process.id) != option.title.text) {
                    first = true
                }
            }
            if (first) {
                first = false;
                option = new Option();
                option.title.text = 'PID-' + process.id;
                for (var j in process.methods) {
                    var method = process.methods[j];
                    option.legend.data.push(method.name);
                    var seriesItem = new SeriesItem();
                    seriesItem.name = method.name;
                    var now = new Date() 
                    value = {
                        name: now.toString(),
                        value: [now, method.qps]
                    }
                    seriesItem.data.push(value);
                    option.series.push(seriesItem);
                }
                chart.setOption(option, true);
            } else {
                for (var j in process.methods) {
                    var method = process.methods[j];
                    var now = new Date();
                    value = {
                        name: now.toString(),
                        value: [now, method.qps]
                    }
                    option.series[j].data.push(value);
                }
                chart.setOption(option, true);
                for (var j in process.methods) {
                    if (option.series[j].data.length == 20) {
                        option.series[j].data.shift()
                    }
                }
            }
        }
    })
}
setInterval(GetStats, 1000)
