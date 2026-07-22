package captcha

import (
	"net/http"
	"net/url"
	"encoding/json"
	"strings"
	"errors"
	"time"
)

var (
	serverURL string
	secretKey string
	ErrFailed = errors.New("captcha verify failed")
	ErrServer = errors.New("captcha server bad status")
	userAgent string
)

var client = &http.Client{
    Timeout: 10 * time.Second,
}

func Set(captchaUrl,key,ua string){
	serverURL=captchaUrl
	secretKey=key
	userAgent=ua
}

func Verify_f(token string) error {
	formData := url.Values{}
	formData.Set("secret", secretKey)
	formData.Set("response", token)
	
	req, err := http.NewRequest("POST",serverURL,strings.NewReader(formData.Encode()))
	
	if err != nil {
		return err
	}
	
	req.Header.Set(
        "Content-Type",
        "application/x-www-form-urlencoded",
    )
	
	if userAgent!="" {
		req.Header.Set("User-Agent",userAgent)
	}
	
	resp, err := client.Do(req)
    if err != nil {
        return err
    }
	
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return ErrServer
	}

	var result struct {
		Success bool `json:"success"`
	}
	
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		resp.Body.Close()
		return err
	}
	resp.Body.Close()

	if result.Success {
		return nil
	} else {
		return ErrFailed
	}
}
