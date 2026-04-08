package domain

type ConnectRequest struct {
	IDInstance       string `json:"idInstance"`
	APITokenInstance string `json:"apiTokenInstance"`
}

type SendMessageRequest struct {
	ConnectRequest
	ChatID  string `json:"chatId"`
	Message string `json:"message"`
}

type SendFileByURLRequest struct {
	ConnectRequest
	ChatID   string `json:"chatId"`
	FileURL  string `json:"fileUrl"`
	FileName string `json:"fileName"`
	Caption  string `json:"caption,omitempty"`
}
