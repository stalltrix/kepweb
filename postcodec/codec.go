package postcodec

import "google.golang.org/protobuf/proto"

type Randtopics struct {
	PostID []string `json:"topicid"`
	Title []string `json:"title"`
	Tag []uint16 `json:"tag"`
	RandRenew int64 `json:"-"`
}

type Reply struct {
    ID   int    `json:"id"`
    User string `json:"user"`
    Meta string `json:"meta"`
    Me   bool   `json:"me"`
    Post string `json:"post"`
    Time int64  `json:"post_time"`
	FirstTime int64 `json:"first_time"`
	Tag  uint16  `json:"tag"`
	Hex string  `json:"hex"`
	MetaTime int64 `json:"-"`
	RandTopic *Randtopics `json:"rand_topic,omitempty"`
}

type Post struct {
    PostHex  string
    TagID    uint16
    Owner    string
    Replies  []Reply
    LastTime int64
	TypeID byte
}

func Encode(p *Post) ([]byte, error) {
    pb := toPB(p)
    return proto.Marshal(pb)
}

func Decode(data []byte) (Post, error) {
    var pbPost PBPost
    if err := proto.Unmarshal(data, &pbPost); err != nil {
        return Post{}, err
    }
    return fromPB(&pbPost), nil
}