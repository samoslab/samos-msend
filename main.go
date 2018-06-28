package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

//Csrf token for next operation
type Csrf struct {
	Csrf_token string
}

//check if file exists
func PathExists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func handlerSender(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	//fmt.Fprintf(w, "path: %s", r.URL.Path)
	//fmt.Fprintf(w, "scheme: %s", r.URL.Scheme)
	wltPath := r.Form["wlt"][0]
	address := r.Form["address"][0]
	fmt.Fprintf(w, "address: %s", address)
	addresses := r.Form["address[]"]
	coins := r.Form["coin[]"]

	if !PathExists(wltPath) {
		fmt.Fprintf(w, "Wallet Path: %s doesn't exist", wltPath)
		return
	}
	if address == "" {
		fmt.Fprintf(w, "address can't be empty", address)
		return
	}

	//to addresses
	toaddresses := make(map[string]int64)
	for k, v := range addresses {
		//	fmt.Fprintln(w, k)
		// fmt.Fprintln(w, v)
		// fmt.Fprintln(w, coins[k])
		value, err := strconv.ParseInt(coins[k], 10, 64)
		if err != nil && coins[k] != "" {
			fmt.Fprintf(w, "Coins num is not corret:%s %s", coins[k], err)
			return
		}
		if value > 0 {
			toaddresses[v] = value
			fmt.Fprintln(w, v)
			fmt.Fprintln(w, coins[k])
		}
	}

	trySend(wltPath, address, toaddresses, w, r)

}

//Transaction struct
type Transaction struct {
	Wallet        map[string]string   `json:"wallet"`
	Hours         map[string]string   `json:"hours_selection"`
	ChangeAddress string              `json:"change_address"`
	To            []map[string]string `json:"to"`
}

//RawTx struct
type RawTx struct {
	Trans interface{} `json:"transaction"`
	RawTx string      `json:"encoded_transaction"`
}

func trySend(wltPath string, address string, toaddresses map[string]int64, w http.ResponseWriter, r *http.Request) {
	//get csrf token first
	url := "http://127.0.0.1:8640/csrf"
	resp, e := http.Get(url)
	if e != nil {
		fmt.Fprintf(w, "There was an error:%s", e)
		return
	}
	defer resp.Body.Close()

	temp, _ := ioutil.ReadAll(resp.Body)
	//fmt.Fprintf(w, "resp:%s", temp)
	var csrf Csrf

	err := json.Unmarshal(temp, &csrf)
	if err != nil {
		fmt.Fprintf(w, "There was an error:%s", err)
	}
	fmt.Fprintf(w, "csrf token value:%s", csrf.Csrf_token)

	//combine the transaction
	var transaction Transaction

	walletMap := make(map[string]string)
	walletMap["id"] = filepath.Base(wltPath)
	transaction.ChangeAddress = address
	transaction.Wallet = walletMap

	selectionMap := make(map[string]string)
	selectionMap["type"] = "auto"
	selectionMap["mode"] = "share"
	selectionMap["share_factor"] = "0.5"
	transaction.Hours = selectionMap

	var tos = []map[string]string{}
	for k, v := range toaddresses {
		item := make(map[string]string)
		item["address"] = k
		item["coins"] = strconv.FormatInt(v, 10)
		tos = append(tos, item)
	}
	transaction.To = tos

	transactionData, err := json.Marshal(transaction)
	if err != nil {
		//panic(err)
	}
	fmt.Fprintf(w, "<br/><br/>Post Transaction Data value:%s", string(transactionData))

	req, err := http.NewRequest("POST", "http://127.0.0.1:8640/wallet/transaction", bytes.NewBuffer(transactionData))
	req.Header.Set("X-CSRF-Token", csrf.Csrf_token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err2 := client.Do(req)
	if err2 != nil {
		panic(err2)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
	if resp.StatusCode == 200 {
		//combine injection transaction
		var rawTx RawTx

		err3 := json.Unmarshal(body, &rawTx)
		if err3 != nil {
			fmt.Fprintf(w, "There was an error:%s", err3)
		}
		//fmt.Fprintf(w, "<br/><br/> Raw Tx:%s", rawTx.RawTx)

		var injectTxt = make(map[string]string)
		injectTxt["rawtx"] = rawTx.RawTx

		injectTransactionData, _ := json.Marshal(injectTxt)

		//injectTransaction
		req, _ := http.NewRequest("POST", "http://127.0.0.1:8640/injectTransaction", bytes.NewBuffer(injectTransactionData))
		req.Header.Set("X-CSRF-Token", csrf.Csrf_token)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err2 := client.Do(req)
		if err2 != nil {
			panic(err2)
		}
		defer resp.Body.Close()

		fmt.Println("injectTransaction response Status:", resp.Status)
		fmt.Println("injectTransaction response Headers:", resp.Header)
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("injectTransaction response Body:", string(body))
		if resp.StatusCode == 200 {

			fmt.Fprintf(w, "\n<br/><br/>injectTransaction success:%s", string(body))
			return

		} else {
			fmt.Fprintf(w, "\n<br/><br/>injectTransaction failed:%s", string(body))
		}

	}
	fmt.Fprintf(w, "\n<br/><br/>injectTransaction failed:")

}

func main() {
	fmt.Println("Sender Coin")
	http.Handle("/", http.FileServer(http.Dir("./")))
	http.HandleFunc("/msend", handlerSender)
	http.ListenAndServe(":8080", nil)

}
