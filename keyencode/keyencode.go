package keyencode

import (
    "crypto/ed25519"
    "crypto/x509"
    "encoding/pem"
	"encoding/base64"
	"bytes"
	"errors"
)

func Key32_encode(raw []byte) ([]byte,error){
	if len(raw)!=32{
		return nil,errors.New("32 key len err")
	}
	pubDer, err := x509.MarshalPKIXPublicKey(ed25519.PublicKey(raw))
    if err != nil {
        return nil,err
    }
    pubPem := pem.EncodeToMemory(&pem.Block{
        Type:  "PUBLIC KEY",
        Bytes: pubDer,
    })
	return pubPem,nil
}

func Key32_decode(data []byte) ([]byte,error){
	pubBlock,rest := pem.Decode(data)
    if pubBlock == nil {
        return nil,errors.New("failed to decode public PEM")
    }
	if len(bytes.TrimSpace(rest)) != 0 {
		 return nil,errors.New("public PEM too many args")
	}
    pubAny, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
    if err != nil {
        return nil,err
    }
    parsedPub, ok := pubAny.(ed25519.PublicKey)
    if !ok {
        return nil,errors.New("not public key")
    }
	if len(parsedPub)!=32{
		return nil,errors.New("key len err")
	}
	return parsedPub,nil
}

func Key64_encode(raw []byte) ([]byte,error){
	if len(raw)!=64{
		return nil,errors.New("64 key len err")
	}
	b64 := base64.StdEncoding.EncodeToString(raw)
    var sb bytes.Buffer
    sb.WriteString("-----BEGIN RAW KEY-----\n")
    for i := 0; i < len(b64); i += 64 {
        end := i + 64
        if end > len(b64) {
            end = len(b64)
        }
        sb.WriteString(b64[i:end])
        sb.WriteString("\n")
    }
    sb.WriteString("-----END RAW KEY-----\n")
	return sb.Bytes(),nil
}

func Key64_decode(data []byte) ([]byte, error) {
    const begin = "-----BEGIN RAW KEY-----"
    const end   = "-----END RAW KEY-----"
	data = bytes.TrimSpace(data)

    if !bytes.HasPrefix(data, []byte(begin)) {
        return nil, errors.New("missing begin header")
    }

    if !bytes.HasSuffix(data, []byte(end)) {
        return nil, errors.New("missing end footer")
    }

    data = bytes.TrimPrefix(data, []byte(begin))
    data = bytes.TrimSuffix(data, []byte(end))

    data = bytes.TrimSpace(data)
    data = bytes.ReplaceAll(data, []byte("\n"), nil)
    data = bytes.ReplaceAll(data, []byte("\r"), nil)
	
    decoded := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
    n, err := base64.StdEncoding.Decode(decoded, data)
    if err != nil {
        return nil, err
    }
	if n!=64{
		return nil,errors.New("key len err")
	}
    return decoded[:n], nil
}

func AutoDecode(data []byte) ([]byte,error){
	ndata := bytes.TrimSpace(data)
    if bytes.HasPrefix(ndata, []byte("-----BEGIN")) && bytes.HasSuffix(ndata, []byte("KEY-----")){
        if bytes.HasPrefix(ndata, []byte("-----BEGIN PUBLIC KEY-----")) {
			return Key32_decode(data)
		} else if bytes.HasPrefix(ndata, []byte("-----BEGIN RAW KEY-----")) {
			return Key64_decode(data)
		} else {
			if len(data)==32||len(data)==64{
				return data,nil
			}
			return nil,errors.New("key type unknown")
		}
    }
    return data,nil
}