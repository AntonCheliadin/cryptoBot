package bybit

type OrderResponseDto struct {
	RetCode int         `json:"ret_code"`
	RetMsg  string      `json:"ret_msg"`
	ExtCode interface{} `json:"ext_code"`
	ExtInfo interface{} `json:"ext_info"`
	Result  struct {
		AccountId    string `json:"accountId"`
		Symbol       string `json:"symbol"`
		SymbolName   string `json:"symbolName"`
		OrderLinkId  string `json:"orderLinkId"`
		OrderId      string `json:"orderId"`
		TransactTime string `json:"transactTime"`
		Price        string `json:"price"`
		OrigQty      string `json:"origQty"`
		ExecutedQty  string `json:"executedQty"`
		Status       string `json:"status"`
		TimeInForce  string `json:"timeInForce"`
		Type         string `json:"type"`
		Side         string `json:"side"`
	} `json:"result"`
}
