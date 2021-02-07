package ecg

import (
  "fmt"
  "net/http"
  "io"
  "io/ioutil"
  "bytes"
  "mime/multipart"
  "encoding/json"
  "strings"
  "os"
)

type Params struct {
  MeterNumber string
  Amount string
  MomoNumber string
  Network string
  Voucher string
  TokenURL string
  ApiURL string
}

type TokenResponse struct {
  AccessToken string `json:"access_token"`
  ExpiresIn uint `json:"expires_in"`
  TokenType string `json:"token_type"`
}

type MeterInfo struct {
  Token string `json:"token"`
  AccountNumber string `json:"accountNumber"`
  MeterNumber string `json:"meterNumber"`
  Name string `json:"name"`
  Address string `json:"address"`
  MeterId string `json:"meterId"`
}

type MeterBalance struct {
  LastTopupAmount float32 `json:"lastTopupAmount"`
  Balance float32 `json:"balance"`
  LastTopupDate uint `json:"lastTopupDate"`
  WeekConsumption uint `json:"weekConsumption"`
  HighestConsumptionDay uint `json:"highestConsumptionDay"`
  MaximumConsumption uint `json:"maximumConsumption"`
  LowestConsumptionDay uint `json:"lowestConsumptionDay"`
  MinimumConsumption uint `json:"minimumConsumption"`
  AverageConsumption uint `json:"averageConsumption"`
}

var (
  tokenURL = os.Getenv("tokenURL")
  ApiURL = os.Getenv("ApiURL")
  username = os.Getenv("username")
  password = os.Getenv("password")
  grantType = os.Getenv("grantType")
  clientId = os.Getenv("clientId")
  clientSecret = os.Getenv("clientSecret")
)

var (
  tokenResponse TokenResponse
  meterInfo MeterInfo
  meterBalance MeterBalance
)

func GetParams(meterNumber string, momoNumber string, network string, voucher string, amount string) *Params {
  return &Params{
    MeterNumber: meterNumber,
    Amount: amount,
    MomoNumber: momoNumber,
    Network: network,
    Voucher: voucher,
    TokenURL: tokenURL,
    ApiURL: ApiURL,
  }
}

func apiRequest(url string, method string, headers map[string]string, payload io.Reader) (error, string) {
  response := ""
	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)
  
  if err != nil {
    return err, response
  }

  for k, v := range headers {
	  req.Header.Add(k, v)
  }

  res, err := client.Do(req)
  if err != nil {
    return err, response
  }

	body, err := ioutil.ReadAll(res.Body)
  if err != nil {
    return err, response
  }
	
  defer res.Body.Close()
	return nil, string(body)
}

func print(data ...interface{}) (n int, err error){
  return fmt.Println(data...)
}

func decodeJSON(data *string, key interface{}) error {
  decoder := json.NewDecoder(strings.NewReader(*data))
  return decoder.Decode(key)
}

func getToken(params *Params) (error, string) {
  headers := map[string]string{}
  payload := &bytes.Buffer{}
  writer := multipart.NewWriter(payload)
  _ = writer.WriteField("username", username)
  _ = writer.WriteField("password", password)
  _ = writer.WriteField("grant_type", grantType)
  _ = writer.WriteField("client_id", clientId)
  _ = writer.WriteField("client_secret", clientSecret)
  err := writer.Close()
  if err != nil {
    return err, ""
  }

  headers["Content-Type"] = writer.FormDataContentType()

  err, res := apiRequest(params.TokenURL, "POST", headers, payload)
  if err != nil {
    return err, ""
  }
  err = decodeJSON(&res, &tokenResponse)

  if err != nil {
    return err, ""
  }

  return nil, tokenResponse.AccessToken
}


func verifyMeter(token *string, params *Params) (error, string){
  headers := map[string]string{}
  headers["Content-Type"] = "application/json"
  headers["Authorization"] = "Bearer " + *token
  payload := strings.NewReader(`{ 
    "MeterNumber": "`+ params.MeterNumber +`",
    "BillingType": 1,
    "AccountNumber": ""
  }`)
  return apiRequest(params.ApiURL + "metermanagement/verify", "POST", headers, payload)
}

func addMeter(token *string, meterInfo *string, params *Params) (error, string) {
  headers := map[string]string{}
  headers["Content-Type"] = "application/json"
  headers["Authorization"] = "Bearer " + *token
  payload := strings.NewReader(*meterInfo)
  return apiRequest(params.ApiURL + "metermanagement/addprepaidmeter", "POST", headers, payload)
}

func getMeterBalance(token *string, meterInfo *MeterInfo, params *Params) (error, string) {
  headers := map[string]string{}
  headers["Content-Type"] = "application/json"
  headers["Authorization"] = "Bearer " + *token
  return apiRequest(params.ApiURL + "Dashboard/Get/" + meterInfo.AccountNumber + "/" + meterInfo.MeterNumber, "GET", headers, nil)
}

func InitGetMeterBalance(params *Params) string {
  err, token := getToken(params)
  
  if err != nil {
    return "Failed to retrieve token"
  }

  err, meterDetails := verifyMeter(&token, params)
  if err != nil {
    return meterDetails
  }

  err = decodeJSON(&meterDetails, &meterInfo)
  if err != nil {
    return meterDetails
  }

  err, info := addMeter(&token, &meterDetails, params)
 
  if err != nil {
    return info
  }

  _, meterBalance := getMeterBalance(&token, &meterInfo, params)
  return meterBalance
}

func makePayment(token *string, meterInfo *MeterInfo, params *Params) (error, string) {
  headers := map[string]string{}
  headers["Content-Type"] = "application/json"
  headers["Authorization"] = "Bearer " + *token
  payload := strings.NewReader(`{
      "MeterId": "` + meterInfo.MeterId + `",
      "AccountNumber": "` + meterInfo.AccountNumber + `",
      "MeterNumber": "` + meterInfo.MeterNumber + `",
      "VoucherNumber": "` + params.Voucher + `",
      "MobileNumber": "` + params.MomoNumber + `",
      "Amount": ` + params.Amount + `,
      "Network": "` + params.Network + `"
    }`)
  return apiRequest(params.ApiURL + "prepaid/makepayment", "POST", headers, payload)
}

func InitMakePayment(params *Params) string {
  err, token := getToken(params)
 
  if err != nil {
    return "Failed to retrieve token"
  }


  err, meterDetails := verifyMeter(&token, params)
  
  if err != nil {
    return meterDetails
  }

  err = decodeJSON(&meterDetails, &meterInfo)

  if err != nil {
    return meterDetails
  }

  err, info := addMeter(&token, &meterDetails, params)

  if err != nil {
    return info
  }
  output := ""

  err, _output  := getMeterBalance(&token, &meterInfo, params)
  if err != nil {
    return _output
  }
  output = _output + "\n"

  err, _output = makePayment(&token, &meterInfo, params)
  if err != nil {
    return _output
  }
  output += _output
  return output
}
