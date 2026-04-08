package domain

type OutboundTextMessage struct {
	ChatID  string
	Message string
}

type OutboundFileMessage struct {
	ChatID   string
	URLFile  string
	FileName string
	Caption  string
}
