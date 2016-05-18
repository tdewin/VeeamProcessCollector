package main
import (
	"os" 
	"io/ioutil"
	"bytes"
	"html/template"
)

type IndexPage struct {
	Naptime int
	Debug int
}
func getIndex(naptime int) string {
	ind := "_inject_index.html"
	if _, err := os.Stat(ind); err == nil {
	  f,err := ioutil.ReadFile(ind)
	  if err == nil {
		return string(f);
	  }
	}

    tpl := `
<!DOCTYPE HTML>
<html lang="en-US">
<head>
	<meta charset="UTF-8">
	<title>Proc Mon</title>
	<script src="/jquery.js"></script>
	<style type="text/css">
		tr.toprow td {
			font-weight: bold;
		}
		#rfilter { width:500px; }
		.visdiv {
			background-color:orange;
			height:10px;
			width:10px;
		}
		.countname {
			font-weight: bold;
			width:150px;
		}
		.countercr {
			width:100px;
		}
	</style>
	<script>
		function mbcalc(strp) {
			return Math.round(parseInt(strp)/(1024*1024)*100)/100
		}
		function cpuc(strp) {
			return Math.round(parseFloat(strp)*100)/100
		}
	    function parseData(xml) {
			serverhtml = ""
			
			filter = false
			testfilter = $("#rfilter").val()
			rfilterobj = 0
			if (testfilter != "") {
				try {
					rfilterobj = new RegExp(testfilter, "i")
					filter = true
				} catch(e) {
					filter = false
				}	
			}
			
			$(xml).find("Server").each(function () {
				servername = $(this).find("ServerName").text()
				divname = servername.replace(/\./g,'_')
				
				debug(servername)
				
				if ($("#serverdiv-"+divname).length == 0) {
					divtxt = "<div id='serverdiv-"+divname+"'><div id='main-serverdiv-"+divname+"'></div><div id='proc-serverdiv-"+divname+"'></div></div>"
					$("#result").append(divtxt)	
				}
				serverhtml = "<h1>"+$(this).find("ServerName").text()+"</h1>"
				var d = new Date(0); // The 0 there is the key, which sets the date to the epoch
				d.setUTCSeconds(parseInt($(this).find("Date").text()));
				serverhtml += d+"<br><table><tr>"
				serverhtml += "<td class='countname'>NET MByte/s</td><td class='countercr'>"+(mbcalc($(this).find("NetBytesPerSec").text()))+"</td>"
				serverhtml += "<td class='countname'>DISK MByte/s</td><td class='countercr'>"+(mbcalc($(this).find("DiskBytesPerSec").text()))+"</td>"
				serverhtml += "<td class='countname'>DISK IO/s</td><td class='countercr'>"+($(this).find("DiskTransfersPerSec").text())+"</td>"
				serverhtml += "<td class='countname'>CORES</td><td class='countercr'>"+$(this).find("Cores").text()+"</td>"
				serverhtml += "</tr></table><br>"
				
				prochtml = "<table><tr class='toprow'><td>PID</td><td>PPID</td><td>Proc</td><td>CPU&#37</td><td>MEM MB</td><td>IO MBs</td><td>CMD</td></tr>"
				$(this).find("VeeamProcess").each(function() {
					goodtogo = true 
					procname = $(this).find("ProcessName").text()
					if (filter) {
						if (!rfilterobj.test(procname)) {
							goodtogo = false
						}
					}
					
					if (goodtogo) {
						prochtml += "<tr>"
						prochtml += "<td style='width:50px;'>"+$(this).find("ProcessID").text()+"</td>"
						prochtml += "<td style='width:50px;'>"+$(this).find("ParentProcessID").text()+"</td>"
						prochtml += "<td style='width:200px;'>"+procname+"</td>"
						prochtml += "<td style='width:80px;'>"+(cpuc($(this).find("CpuPct").text()))+""+"</td>"
						prochtml += "<td style='width:80px;'>"+(mbcalc($(this).find("WorkingSetPrivate").text()))+""+"</td>"
						prochtml += "<td style='width:80px;'>"+(mbcalc($(this).find("IOBytesPerSec").text()))+""+"</td>"
						prochtml += "<td>"+$(this).find("CommandLine").text()+"</td>"
						prochtml += "</tr>"
					}
				});
				prochtml += "</table>"
				$("#main-serverdiv-"+divname).html(serverhtml)
				$("#proc-serverdiv-"+divname).html(prochtml)
			 });
			 
		}

		function debug(txt) {
			if ({{.Debug}}) {
				console.log(txt)
			}
		}
		function refresh() {
			if ($("#refresh").prop('checked')) {
				$.ajax({
					type: "GET",
					url: "http://localhost:46101/xml",
					dataType: "xml",
					success: parseData
				   });
			}
		}
		function foreverRefresh() {
			refresh();
			debug("Refreshing");
			setTimeout(function() { foreverRefresh() }, {{.Naptime}});
		}
		
		$("document").ready(function() {
			foreverRefresh();
		});
	</script>
</head>
<body>
	<input type="checkbox" checked id="refresh"/>Refresh<br>
	Regex Process Filter : <input type="input" id="rfilter" value=""/>
	<div id="result">
	</div>
</body>
</html>
`
t, err := template.New("Main").Parse(tpl)
if err == nil {
	var b bytes.Buffer
	err = t.Execute(&b, IndexPage{Naptime:naptime*1000,Debug:1})
	if err == nil {
		return b.String()
	} else {
		return "Internal Error, template is not good";
	}
} else {
	 return "Internal Error, template is not good";
}
	

}