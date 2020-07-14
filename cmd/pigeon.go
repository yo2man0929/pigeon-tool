package cmd

// Resultdata is the structure of host list api
type Resultdata struct {
	Members []string `json:"members"`
}

// Subscriptions is the innerest layer of pigeon status api struct
type Subscriptions struct {
	TopicName        string   `json:"topicName"`
	Property         string   `json:"property"`
	OldMessageCount  int      `json:"oldMessageCount"`
	OldMessages      []string `json:"oldMessages"`
	SubscriptionName string   `json:"subscriptionName"`
}

// OutSub is the upper layer of Subscriptions
type OutSub struct {
	Sub []Subscriptions `json:"subscriptions"`
}

//Outmost is the upper of OutSub
type Outmost struct {
	PigeonStatus OutSub `json:"pigeonStatus"`
	Host         string `json:"host"`
}

// Information declared as used variable
type Information struct {
	pigeonHostEndpoint string
	StatusURL          string
	SkipURL			   string
	cert               string
	HostList           []Resultdata
	StatusResult       Outmost
	tailCount          int
}
