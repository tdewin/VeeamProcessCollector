package main


func getIndex() string {
return `
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
	</style>
	<script>
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
				
				if ($("#serverdiv-"+servername).length == 0) {
					$("#result").append("<div id='serverdiv-"+servername+"'><div>")
				}
				serverhtml = "<h1>"+$(this).find("ServerName").text()+"</h1>"
				var d = new Date(0); // The 0 there is the key, which sets the date to the epoch
				d.setUTCSeconds(parseInt($(this).find("Date").text()));
				serverhtml += ""+d+"<br><table><tr class='toprow'><td>Proc</td><td>CPU&#37</td><td>MEM MB</td><td>CMD</td></tr>"
				$(this).find("VeeamProcess").each(function() {
					goodtogo = true 
					procname = $(this).find("ProcessName").text()
					if (filter) {
						if (!rfilterobj.test(procname)) {
							goodtogo = false
						}
					}
					
					if (goodtogo) {
						serverhtml += "<tr><td style='width:200px;'>"+procname+"</td>"
						serverhtml += "<td style='width:80px;'>"+$(this).find("CpuPct").text()+""+"</td>"
						serverhtml += "<td style='width:80px;'>"+Math.round(parseInt($(this).find("WorkingSetPrivate").text())/(1024*1024)*100)/100+""+"</td>"
						serverhtml += "<td>"+$(this).find("CommandLine").text()+"</td>"
						serverhtml += "</tr>"
					}
				});
				serverhtml += "</table><br><br>"
				$("#serverdiv-"+servername).html(serverhtml)
			 });
			 
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
			setTimeout(function() { foreverRefresh() }, 5000);
		}
		
		$("document").ready(function() {
			foreverRefresh();
		});
	</script>
</head>
<body>
	<input type="checkbox" id="refresh"/>Refresh<br>
	Regex Process Filter : <input type="input" id="rfilter" value=""/>
	<div id="result">
	</div>
</body>
</html>
`}