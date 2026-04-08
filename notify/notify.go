package notify

import (
	"time"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"github.com/stalltrix/kep-demo/logger"
)

type _regdFS struct {
    tag     int
    lastmod int64
}

var (
	sys_init bool
	done_lock bool
	regdFS []_regdFS
	BaseDir = "kep-data"
	callbackMap map[int]func(int)
	log logger.Log_TYPE
)

func Init_path(self string){
	if sys_init {
		return
	}
	sys_init=true
	if self != "" {
        BaseDir = filepath.Join(self, "kep-data")
    }
	callbackMap=make(map[int]func(int))
	log.SetLevel("debug")
}

func check_fs(){
for{
	time.Sleep(time.Second * 40)
	for i:=range regdFS {
		idxPath := filepath.Join(BaseDir, "tag_"+strconv.Itoa(regdFS[i].tag)+".idx")
		info, err := os.Stat(idxPath)
		if err !=nil {
			log.Println("check idx err:",err)
			continue
		}
		modtime := info.ModTime().Unix()
		if modtime!=regdFS[i].lastmod {
			regdFS[i].lastmod=modtime
			function,ok:=callbackMap[regdFS[i].tag]
			if ok {function(regdFS[i].tag);}
		}
	}
}
}

func Reg_fs(fs_tag int, callback func(int)) error {
	if !sys_init {
		return errors.New("notify not init")
	}
	if done_lock {
		return errors.New("done lockd")
	}
	_,ok:=callbackMap[fs_tag]
	if ok {
		return errors.New("reg exist")
	}
	idxPath := filepath.Join(BaseDir, "tag_"+strconv.Itoa(fs_tag)+".idx")
	info, err := os.Stat(idxPath)
	if err != nil {
		return err
	}
	regdFS=append(regdFS,_regdFS{
		tag: fs_tag,
		lastmod: info.ModTime().Unix(),
	})
	callbackMap[fs_tag]=callback
	return nil
}

func Done(){
	if done_lock {
		return
	}
	done_lock=true
	go check_fs()
}