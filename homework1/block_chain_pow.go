package main

// transation交易结构
type Transation struct {
	Sender  string `json:"sender"`
	Recipient string `json:"recipient"`
	Amount int `json:"amount"`
}

// Block区块结构
type Block struct{
	Index	int		`json:"index"`
	Timestamp	int64	`json:"timestamp"`
	Transations []Transation `json:"transactions"`
	Proof	int64 `json:"proof"`
	PreviousHash string `json:"provious_hash"`
}

//Blockchain区块链
type Blockchain struct{
	Chain []Block
	CurrentTransations []Transation
	Difficulty int
}