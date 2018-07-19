package stats

import (
	"bytes"
	"encoding/json"
	"os"
	"sync"
)

var (
	gStats *stats
	once   sync.Once
)

func GlobalStats() *stats {
	once.Do(func() {
		gStats = newStats()
	})
	return gStats
}

type stats struct {
	sync.Mutex
	Id     int            `json:"id"`
	QPS    int            `json:"qps"`
	Method map[string]int `json:"method"`
	Html   string         `json:"-"`
	data   []byte
}

func newStats() *stats {
	var buf bytes.Buffer
	buf.WriteString(statsBeginHTML)
	buf.WriteString(statsJS)
	buf.WriteString(statsEndHTML)

	return &stats{
		Id:     os.Getpid(),
		Method: make(map[string]int),
		Html:   buf.String(),
	}
}

func (s *stats) Calculate() {
	s.Lock()
	defer s.Unlock()

	for _, qps := range s.Method {
		s.QPS += qps
	}
	s.data, _ = json.Marshal(s)

	// reset
	s.QPS = 0
	for name, _ := range s.Method {
		s.Method[name] = 0
	}
}

func (s *stats) Bytes() []byte {
	s.Lock()
	defer s.Unlock()

	return s.data
}

const statsJS = `
var first=true;var chart=echarts.init(document.getElementById("chart-0"));var Option=function(){return{title:{text:""},tooltip:{trigger:"axis",formatter:function(params){params=params[0];var date=new Date(params.name);return date.getDate()+"/"+(date.getMonth()+1)+"/"+date.getFullYear()+" : "+params.value[1]},axisPointer:{animation:false}},legend:{data:[]},xAxis:{type:"time",splitLine:{show:false}},yAxis:{type:"value",axisLabel:{formatter:"{value} qps"},boundaryGap:[0,"100%"],splitLine:{show:false}},series:[]}};var option=new Option;var SeriesItem=function(){return{name:"",type:"line",showSymbol:false,hoverAnimation:false,smooth:true,data:[],markPoint:{data:[{type:"max",name:"最大值"},{type:"min",name:"最小值"}]},markLine:{data:[{type:"average",name:"平均值"}]}}};var ProcessStats=function(){return{id:0,methods:[]}};var MethodStats=function(){return{name:"",qps:-1}};function Convert(data){var process=new ProcessStats;process.id=data.id;var method=new MethodStats;method.name="Total";method.qps=data.qps;process.methods.push(method);for(var name in data.method){var method=new MethodStats;method.name=name;method.qps=data.method[name];process.methods.push(method)}return process}function GetStats(){jQuery.ajax({url:"http://##ListenAddr##/stats",type:"get",async:"true",dataType:"json",success:function(data){var process=Convert(data);if(!first){if(process.methods.length!=option.series.length){first=true}else if("PID-"+process.id!=option.title.text){first=true}}if(first){first=false;option=new Option;option.title.text="PID-"+process.id;for(var j in process.methods){var method=process.methods[j];option.legend.data.push(method.name);var seriesItem=new SeriesItem;seriesItem.name=method.name;var now=new Date;value={name:now.toString(),value:[now,method.qps]};seriesItem.data.push(value);option.series.push(seriesItem)}chart.setOption(option,true)}else{for(var j in process.methods){var method=process.methods[j];var now=new Date;value={name:now.toString(),value:[now,method.qps]};option.series[j].data.push(value)}chart.setOption(option,true);for(var j in process.methods){if(option.series[j].data.length==20){option.series[j].data.shift()}}}}})}setInterval(GetStats,1e3);
`

const statsBeginHTML = `
<!DOCTYPE html><html style="height: 100%"><head><meta charset="utf-8"><title>Box性能监控</title></head><body style="height: 100%; margin: 0"><div id="chart-0"style="height:400px"></div><div id="chart-1"style="height:400px"></div><div id="chart-2"style="height:400px"></div><script type="text/javascript"src="http://echarts.baidu.com/gallery/vendors/echarts/echarts.min.js"></script><script type="text/javascript"src="http://echarts.baidu.com/gallery/vendors/echarts-gl/echarts-gl.min.js"></script><script type="text/javascript"src="http://echarts.baidu.com/gallery/vendors/echarts-stat/ecStat.min.js"></script><script type="text/javascript"src="http://echarts.baidu.com/gallery/vendors/echarts/extension/dataTool.min.js"></script><script type="text/javascript"src="http://echarts.baidu.com/gallery/vendors/echarts/map/js/china.js"></script><script type="text/javascript"src="http://echarts.baidu.com/gallery/vendors/echarts/map/js/world.js"></script><script type="text/javascript"src="http://api.map.baidu.com/api?v=2.0&ak=ZUONbpqGBsYGXNIYHicvbAbM"></script><script type="text/javascript"src="http://echarts.baidu.com/gallery/vendors/echarts/extension/bmap.min.js"></script><script type="text/javascript"src="http://echarts.baidu.com/gallery/vendors/simplex.js"></script><script type="text/javascript"src="http://code.jquery.com/jquery-latest.js"></script><script type="text/javascript">
`

const statsEndHTML = `
</script></body></html>
`
