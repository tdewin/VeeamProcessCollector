package main

import (
	"fmt"
	"net/http"
	"strings"
	"flag"
	"log"
	"time"
	"math/rand"
	"encoding/xml"
	"sync"
)
type UpdatePost struct {
	Date uint32 `xml:"Date"`
	Cn uint64 `xml:"Cn"`
	Server string`xml:"Server"`
	VeeamProcesses []*VeeamProcess `xml:"VeeamProcesses>VeeamProcess"`
	VeeamServerStat *VeeamServerStat  `xml:"Stats"`
}
type VeeamProcess struct {
	ProcessName string `xml:"ProcessName"`
	CommandLine string`xml:"CommandLine"`
	ExecutablePath string`xml:"ExecutablePath"`
	ProcessID uint32 `xml:"ProcessID"`
	ParentProcessID uint32	`xml:"ParentProcessID"`	
	Stats *VeeamProcessStat `xml:"Stats"`	
}
type VeeamProcessStat struct {
	IOBytesPerSec uint64 `xml:"IOBytesPerSec"`
	WorkingSetPrivate uint64 `xml:"WorkingSetPrivate"`
	CpuPct float32 `xml:"CpuPct"`
}
type VeeamServerStat struct {
	NetBytesPerSec uint64 `xml:"NetBytesPerSec"`
	DiskBytesPerSec uint64 `xml:"DiskBytesPerSec"`
	Cores uint `xml:"Cores"`
}
type UpdateQueue struct {
	lock sync.Mutex
	updates []*UpdatePost
}
type InfraView struct {
	lock sync.RWMutex
	serverviews ServerViews `xml:"Servers"`
}
type ServerView struct {
	Date uint32 `xml:"Date"`
	Cn uint64 `xml:"Cn"`
	Server string`xml:"ServerName"`
	VeeamProcesses []*VeeamProcess `xml:"VeeamProcesses>VeeamProcess"`
	VeeamServerStat  *VeeamServerStat `xml:"Stats"`
}
//unmarshaller for maps http://stackoverflow.com/questions/30928770/marshall-map-to-xml-in-go
type ServerViews map[string]*ServerView
func (s ServerViews) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	serversstart := xml.StartElement{Name: xml.Name{"","Servers"}}
	e.EncodeToken(serversstart)
	
    for _, value := range s {
		serverstart := xml.StartElement{Name: xml.Name{"","Server"}}
		serverstart.Attr = append(serverstart.Attr,xml.Attr{xml.Name{"","Name"},value.Server})
        e.EncodeElement(value,serverstart)
    }

	e.EncodeToken(xml.EndElement{serversstart.Name})

    // flush to ensure tokens are written
    err := e.Flush()
    if err != nil {
        return err
    }

    return nil
}

type Answer struct {
	Error string `xml:"Error,omitempty"`
	Success string `xml:"Success,omitempty"`
	Cn uint64 `xml:"Cn,omitempty"`
}

type VeeamProcessCollector struct {
	key string
	stop bool
	rwlock sync.RWMutex
	uq *UpdateQueue
	iv *InfraView
	naptime int
}
func (h *VeeamProcessCollector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path,"/")
	action := strings.Split(path,"/")
	
	if path == "" {
		fmt.Fprintf(w,getIndex(h.naptime))
	} else {
		switch action[0] {
			case "xml":
				h.iv.lock.RLock()
				defer h.iv.lock.RUnlock()
				output, err := xml.MarshalIndent(h.iv.serverviews, "  ", "    ")
				if err == nil {
					fmt.Fprintf(w,"%s",output);
				} else {
					fmt.Fprintf(w,"<Servers><error>%s</error></Servers>",err);
				}
				
			break;
			case "jquery.js":
				fmt.Fprintf(w,"%s",getJquery());
			break
			case "cnquery":
				r.ParseForm()
				h.rwlock.RLock()
				defer h.rwlock.RUnlock()
				if r.Form.Get("key") == h.key  {
					varserv := r.Form.Get("server")
					if varserv != "" {
						h.iv.lock.RLock()
						defer h.iv.lock.RUnlock()
						server := h.iv.serverviews[varserv]
						if server != nil {
							answer,_ := xml.Marshal(Answer{Success:"OK",Cn:(server.Cn+50)})
							fmt.Fprintf(w,"%s\r\n%s",xml.Header,answer)
						} else {
							answer,_ := xml.Marshal(Answer{Success:"OK",Cn:(50)})
							fmt.Fprintf(w,"%s\r\n%s",xml.Header,answer)
						}
					} else {
						answer,_ := xml.Marshal(Answer{Error:"No server given"})
						fmt.Fprintf(w,"%s\r\n%s",xml.Header,answer)
					}
				} else {
					answer,_ := xml.Marshal(Answer{Error:"Key Invalid"})
					fmt.Fprintf(w,"%s\r\n%s",xml.Header,answer)
					log.Printf("Rogue agent requesting Cn : %s , key %s",r.RemoteAddr,r.Form.Get("key"))				
				}
			break;
			case "postproc":
				h.rwlock.RLock()
				defer h.rwlock.RUnlock()
				log.Print("Incoming")
				r.ParseForm()
				if r.Form.Get("key") == h.key  {
					if r.Form.Get("update") != "" {

						
						update := UpdatePost{}
						
						strreader := strings.NewReader(r.Form.Get("update"))
						decoder := xml.NewDecoder(strreader)
						err := decoder.Decode(&update)
						
						if err == nil {
							if h.stop {
								answer,_ := xml.Marshal(Answer{Success:"STOP"})
								fmt.Fprintf(w,"%s\r\n%s",xml.Header,answer)
							} else {
								answer,_ := xml.Marshal(Answer{Success:"OK"})
								fmt.Fprintf(w,"%s\r\n%s",xml.Header,answer)
							}


							log.Printf("Update from %s",update.Server)
							
							//add to the queue in a separate thread so http request returns
							go func(h *VeeamProcessCollector,update *UpdatePost) {
								h.uq.lock.Lock()
								defer h.uq.lock.Unlock()
								h.uq.updates = append(h.uq.updates,update)
								log.Printf("Added, updates in queue : %d",len(h.uq.updates))
							} (h,&update)
							
						} else {
							answer,_ := xml.Marshal(Answer{Error:"FORMAT"})
							fmt.Fprintf(w,"%s\r\n%s",xml.Header,answer)
							log.Printf("Could not parse incoming %s",r.Form.Get("update"))
							log.Print(err)
						}
					} else {
						fmt.Fprintf(w,"Empty update")
					}
				} else {
						fmt.Fprintf(w,"Key invalid")
						log.Printf("Rogue agent posting : %s",r.RemoteAddr)
				}
				
			break; 
			case "stop":
				h.rwlock.Lock()
				h.stop = true
				h.rwlock.Unlock()
			break;
		}
	}
}


func transform(h* VeeamProcessCollector) {
	h.uq.lock.Lock()
	defer h.uq.lock.Unlock()
	numupd := len(h.uq.updates)
	if numupd  > 0 {
		log.Printf("Processing %d",numupd);
		h.iv.lock.Lock()
		defer h.iv.lock.Unlock()
		
		for u := (numupd-1); u >=0;u-- {
			upd := h.uq.updates[u]
			
			if h.iv.serverviews[upd.Server] == nil {
				sv := ServerView{upd.Date,upd.Cn,upd.Server,upd.VeeamProcesses,upd.VeeamServerStat}
				h.iv.serverviews[upd.Server] = &sv
			} else if h.iv.serverviews[upd.Server].Cn < upd.Cn  {
				h.iv.serverviews[upd.Server].Date = upd.Date
				h.iv.serverviews[upd.Server].Cn = upd.Cn
				h.iv.serverviews[upd.Server].VeeamProcesses = upd.VeeamProcesses
				h.iv.serverviews[upd.Server].VeeamServerStat  = upd.VeeamServerStat 
			} else {
				log.Printf("Old update, ignoring, collector might not be able to hand load");
			}
		}
		h.uq.updates = []*UpdatePost{}
	} else {
		log.Printf("No updates");
	}
}
func transformLoop(h* VeeamProcessCollector) {
	for {
		transform(h)
		time.Sleep(time.Duration(h.naptime) * time.Second)
	}
}

func main() {
    fport := flag.Int("port", 46101, "Specify port listening")
    fkey := flag.String("key", "", "key for posting")
    fnaptime := flag.Int("naptime", 3, "Nap time")
	
    flag.Parse()
	
	key := *fkey
	//http://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang
	if key == "" {
		rand.Seed(time.Now().UnixNano())
		var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz0123456789")
		b := make([]rune, 10)
		
		for i := range b {
			b[i] = letterRunes[rand.Intn(len(letterRunes))]
		}
		key = string(b)
	}
	uq := UpdateQueue{}
	iv := InfraView{}
	iv.serverviews = make(map[string]*ServerView)
    collector := VeeamProcessCollector{key:key,stop:false,uq:&uq,iv:&iv,naptime:*fnaptime}
    go transformLoop(&collector)
	http.Handle("/", &collector)
	
	
	log.Printf("Starting on http://localhost:%d",*fport)
	log.Printf("Start agents with %s %s %d","localhost",key,*fport)
    http.ListenAndServe(fmt.Sprintf(":%d",(*fport)), nil)
}