package meta

import (
	"sync"
	"net"
	"time"
	"errors"
	"net/url"
	"html"
	"encoding/json"
	"strings"
	"golang.org/x/net/idna"
)

var (
	metaMap sync.Map
)

type User_meta struct {
	meta_data string
	lasttime int64
}

func NewTTLMap(){
for {
	time.Sleep(time.Second *60*60*12)
	var newMap sync.Map
	metaMap=newMap
}
}


func dnsLookup(domain string) (string,string,error) {
	txtRecords, err := net.LookupTXT(domain)
    if err != nil {
        return "","",err
    }
	
	var img,name string
	
	for _, txt := range txtRecords {
		if len(txt) >= 4 && txt[:4] == "img=" {
			img=txt[4:]
			continue;
		}
		if len(txt) >= 4 && txt[:4] == "nme=" {
			name=txt[4:]
			continue;
		}
    }
	if img==""&&name=="" {
		return "","",errors.New("nslookup meta not found.")
	}
	return img,name,nil
}

func Meta_get(domain,hexid string) (string,error) {
	val,ok:=metaMap.Load(domain)
	if ok {
		meta_str:=val.(*User_meta)
		if meta_str.lasttime+3600 < time.Now().Unix() {
			meta_data,err:=meta_renew(domain,hexid,true)
			if err!=nil{
				return "",err
			}
			return meta_data,nil
		}else{
			return meta_str.meta_data,nil
		}
	}else{
		meta_data,err:=meta_renew(domain,hexid,false)
		if err!=nil{
			return "",err
		}
		return meta_data,nil
	}
}

func meta_renew(domain,hexid string,is_exist bool) (string,error) {
	img,name,err:=dnsLookup(domain)
	if err != nil {
	if is_exist {
		val,ok:=metaMap.Load(domain)
		if ok {
		meta_str:=val.(*User_meta)
		meta_str.meta_data=""
		meta_str.lasttime=time.Now().Unix()
		}
	} else {
		new_data:=&User_meta{
			meta_data: "",
			lasttime: time.Now().Unix(),
		}
		metaMap.Store(domain,new_data)
	}
		return "",err
	}
	if len(img)>255||len(name)>255{
	if is_exist {
		val,ok:=metaMap.Load(domain)
		if ok {
		meta_str:=val.(*User_meta)
		meta_str.meta_data=""
		meta_str.lasttime=time.Now().Unix()
		}
	} else {
		new_data:=&User_meta{
			meta_data: "",
			lasttime: time.Now().Unix(),
		}
		metaMap.Store(domain,new_data)
	}
		return "",errors.New("meta data too long")
	}
	if img!=""{
		img="https://"+url.QueryEscape(img)
		img=strings.ReplaceAll(img, "%2F", "/")
	}
	if name!=""{
		 unicode, err := idna.ToUnicode(name)
		if err == nil {
			name=unicode
		}
		name=html.EscapeString(name)
	}
	type User struct {
		Name string `json:"name"`
		Img string `json:"img"`
		Id string `json:"id"`
	}
	data := User{
        Name: name,
        Img:  img,
		Id: hexid,
    }
	b, _ := json.Marshal(data)
	
	if is_exist {
		val,ok:=metaMap.Load(domain)
		if ok {
		meta_str:=val.(*User_meta)
		meta_str.meta_data=string(b)
		meta_str.lasttime=time.Now().Unix()
		}
	} else {
		new_data:=&User_meta{
			meta_data: string(b),
			lasttime: time.Now().Unix(),
		}
		metaMap.Store(domain,new_data)
	}
	return string(b),nil
}