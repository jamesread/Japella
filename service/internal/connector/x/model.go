package x

type Tweet struct {
	Text  string      `json:"text"`
	Media *TweetMedia `json:"media,omitempty"`
}

type TweetMedia struct {
	// media_ids: use the media_key string from POST /2/media/upload response
	MediaIds []string `json:"media_ids"`
}

type TweetData struct {
	ID string `json:"id"`
}

type TweetResult struct {
	Data TweetData `json:"data"`
}

type WhoamiResult struct {
	Data WhoamiData `json:"data"`
}

type WhoamiData struct {
	ID        string `json:"id"`
	Namespace string `json:"name"`
	Username  string `json:"username"`
}
