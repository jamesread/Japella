package x

type Tweet struct {
	Text string `json:"text"`
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
