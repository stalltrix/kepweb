package postcodec

func toPB(p *Post) *PBPost {
    pb := &PBPost{
        PostHex:  p.PostHex,
        TagId:    uint32(p.TagID),
        Owner:    p.Owner,
        LastTime: p.LastTime,
        TypeId:   uint32(p.TypeID),
    }

    pb.Replies = make([]*PBReply, len(p.Replies))

    for i, r := range p.Replies {
        pb.Replies[i] = &PBReply{
            Id:       int32(r.ID),
            User:     r.User,
            Meta:     r.Meta,
            Me:       r.Me,
            Post:     r.Post,
            Time:     r.Time,
            Tag:      uint32(r.Tag),
            Hex:      r.Hex,
            MetaTime: r.MetaTime,
        }
    }

    return pb
}

func fromPB(pb *PBPost) Post {
    p := Post{
        PostHex:  pb.PostHex,
        TagID:    uint16(pb.TagId),
        Owner:    pb.Owner,
        LastTime: pb.LastTime,
        TypeID:   byte(pb.TypeId),
    }

    p.Replies = make([]Reply, len(pb.Replies))

    for i, r := range pb.Replies {
        p.Replies[i] = Reply{
            ID:       int(r.Id),
            User:     r.User,
            Meta:     r.Meta,
            Me:       r.Me,
            Post:     r.Post,
            Time:     r.Time,
            Tag:      uint16(r.Tag),
            Hex:      r.Hex,
            MetaTime: r.MetaTime,
        }
    }

    return p
}