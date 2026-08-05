package profile

import (
	"net/http"
	"net/url"
	"bytes"
	"time"
	"encoding/hex"
	"encoding/json"
	"errors"
	"encoding/binary"
	"github.com/stalltrix/kepweb/meta"
	"github.com/stalltrix/kep-demo/kepresolv"
)

type Posts struct {
	Hex string `json:"hex"`
	Title string `json:"title"`
	Time int64 `json:"time"`
}

type UserProfile struct {
    UserID   string    `json:"user_id"`
	UserK    [32]byte  `json:"-"`
	Domain   string    `json:"domain"`
    Username string    `json:"username"`
	Avatar   string    `json:"avatar"`
	Lastupdate int64   `json:"-"`
    Score    int       `json:"score"`
    Topics   []Posts   `json:"topics"`
    Replies  []Posts   `json:"replies"`
}


var (
	Default UserProfile
	userPages []byte
	ErrNull=errors.New("null")
	ErrEnd=errors.New("END")
)

func LoadPage(pageData []byte) {
	userPages=pageData
}

func AddPts(dat kepresolv.Kdata,profile *UserProfile) error {
	if profile == nil {
		return ErrNull
	}
	if profile.UserID=="" {
		return ErrEnd
	}
	var mainkey [32]byte
	binary.BigEndian.PutUint64(mainkey[:], dat.Akey_des)
	if !bytes.Equal(mainkey[:8],profile.UserK[:8]) {
		return ErrEnd
	}
	
	postid:=""
	title:=""
 if dat.Apoint_to==nil {
	postid=hex.EncodeToString(dat.Athash)
	idx := bytes.IndexByte(dat.Atxt, '\n')
	if idx <4 {
		if len(dat.Atxt)>2 {
			if dat.Atxt[0]=='#'&&dat.Atxt[1]==' ' {
				title=string(dat.Atxt[2:])
			}
		}
	} else {
		if len(dat.Atxt)>2 {
			if dat.Atxt[0]=='#'&&dat.Atxt[1]==' ' {
				title=string(dat.Atxt[2:idx])
			}
		}
	}
	if title=="" {
		title="无标题"
	}
 } else {
		postid=hex.EncodeToString(dat.Apoint_to)
		replys := []rune(string(dat.Atxt))
		if len(replys) <= 28 {
			title="Re: "+string(dat.Atxt)
		} else {
			title="Re: "+string(replys[:25]) + "..."
		}
 }
	if dat.Apoint_to==nil {
		//topic
		profile.Score+=3
		profile.Topics=append(profile.Topics,Posts{
			Hex: postid,
			Title: title,
			Time: dat.Atimestamp,
		})
	} else {
		profile.Score++
		profile.Replies=append(profile.Replies,Posts{
			Hex: postid,
			Title: title,
			Time: dat.Atimestamp,
		})
	}
	return nil
}

func PagesHandler(w http.ResponseWriter, r *http.Request,profile *UserProfile) {
	if profile == nil {
		w.WriteHeader(404)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write(userPages)
}

func APIHandler(w http.ResponseWriter, r *http.Request,profile *UserProfile) {
	if profile == nil {
		w.WriteHeader(404)
		return
	}
	now:=time.Now().Unix()
	if profile.Lastupdate+3600 < now {
		profile.Lastupdate=now
		metaData,err:=meta.Meta_get(profile.Domain)
		if err != nil {
			metaData="https://avatar.stalltrix.com/avatar/"+url.QueryEscape(profile.Domain)+".svg"
			profile.Username=profile.Domain
		} else {
			var ImgData struct {
				Name string `json:"name"`
				Img  string `json:"img"`
			}
			err = json.Unmarshal([]byte(metaData), &ImgData)
			if err != nil {
				metaData="https://avatar.stalltrix.com/avatar/"+url.QueryEscape(profile.Domain)+".svg"
				profile.Username=profile.Domain
			} else {
				if ImgData.Name==""{
					ImgData.Name=profile.Domain
				}
				profile.Username=ImgData.Name
				if ImgData.Img==""{
					metaData="https://avatar.stalltrix.com/avatar/"+url.QueryEscape(ImgData.Name)+".svg"
				} else {
					metaData=ImgData.Img
				}
			}
		}
		profile.Avatar=metaData
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(*profile)
}