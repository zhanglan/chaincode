package main

import (
	"bytes"
	"crypto/md5"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"strconv"
	"strings"
	"time"
)

type UniqueCodeChaincode struct {
}

const ADMIN string = "Admin@org1.chains.cloudchain.cn"
const PRIVATE_DATA_COLLECTION_NAME string = "UNIQUE_CODE_SECRET"
const LAST_CODE_ID string = "LAST_CODE_ID"

func (t *UniqueCodeChaincode) Init(stub shim.ChaincodeStubInterface) peer.Response {
	return shim.Success(nil)
}

func (t *UniqueCodeChaincode) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	fn, args := stub.GetFunctionAndParameters()
	if fn != "invoke" {
		return shim.Error(fmt.Sprintf("UniqueCodeChaincode unknown function call fn-->%s--args[0]-->%s", fn, args[0]))
	}
	method := args[0]
	var err error
	var result string
	if method == "createCode" {
		result, err = createCode(stub, args)
	} else if method == "listCode" {
		result, err = listCode(stub, args)
	} else if method == "checkCode" {
		result, err = checkCode(stub, args)
	} else if method == "getNextCodeId" {
		result, err = getNextCodeId(stub)
	} else if method == "setLastCodeId" {
		result, err = setLastCodeId(stub, args)
	}

	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success([]byte(result))
}

func getNextCodeId(stub shim.ChaincodeStubInterface) (string, error) {
	codeId, err := getCurrentCodeId(stub)
	if err != nil {
		return "{\"success\": false, \"msg\":\"get codeId wrong\"}", err
	}
	return fmt.Sprintf("{\"success\": true, \"msg\":\"获取成功\", \"data\": {\"next_code_id\": %s}}", codeId), nil
}

func setLastCodeId(stub shim.ChaincodeStubInterface, args []string) (string, error) {
	codeId := args[1]
	err := stub.PutState(LAST_CODE_ID, []byte(codeId))
	if err != nil {
		return "{\"success\": false, \"msg\":\"putstate error\"}", nil
	}
	return "{\"success\": true, \"msg\":\"设置成功\"}", nil
}

func createCode(stub shim.ChaincodeStubInterface, args []string) (string, error) {
	creator := getCreator(stub)
	creator = getUserId(creator)
	if creator != ADMIN {
		return fmt.Sprintf("{\"success\": false, \"msg\":\"have no authority %s\"}", creator), nil
	}
	codeId, err := getCurrentCodeId(stub)
	if err != nil {
		return "{\"success\": false, \"msg\":\"get codeId wrong\"}", err
	}
	timestampStr := args[1]
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return "{\"success\": false, \"msg\":\"timestamp param format wrong\"}", err
	}
	now := time.Now().UnixNano() / 1e6
	diff := intAbs(now - timestamp)
	if diff > 200000000 {
		return "{\"success\": false, \"msg\":\"timestamp param wrong\"}", nil
	}

	value, err := stub.GetState(codeId)
	if value != nil {
		return "{\"success\": false, \"msg\":\"code has created\"}", nil
	}

	txId := stub.GetTxID()
	secret := createSecret(strconv.FormatInt(timestamp, 10), txId)

	err = stub.PutPrivateData(PRIVATE_DATA_COLLECTION_NAME, codeId, []byte(secret))
	if err != nil {
		return "{\"success\": false, \"msg\":\"save private data error\"}", err
	}
	codeIdIndex, _ := strconv.ParseInt(codeId, 10, 64)
	start := codeIdIndex*10000 + 1
	end := codeIdIndex*10000 + 10000
	codeP := getOrderCode(start) + "-" + getOrderCode(end)
	err = stub.PutState(codeId, []byte(codeP))
	err = stub.PutState(LAST_CODE_ID, []byte(codeId))
	if err != nil {
		return "{\"success\": false, \"msg\":\"putstate error\"}", nil
	}
	return "{\"success\": true, \"msg\":\"create code success\"}", nil
}

func listCode(stub shim.ChaincodeStubInterface, args []string) (string, error) {
	creator := getCreator(stub)
	creator = getUserId(creator)
	if creator != ADMIN {
		return fmt.Sprintf("{\"success\": false, \"msg\":\"have no authority %s\"}", creator), nil
	}

	codeId := args[1]
	value, err := stub.GetState(codeId)
	if value == nil {
		return "{\"success\": false, \"msg\":\"code has not created\"}", nil
	}
	if err != nil {
		return "{\"success\": false, \"msg\":\"get data from chain error\"}", err
	}
	secret, err := stub.GetPrivateData(PRIVATE_DATA_COLLECTION_NAME, codeId)
	if err != nil {
		return "{\"success\": false, \"msg\":\"get secret from private data error\"}", err
	}
	codeIdIndex, _ := strconv.ParseInt(codeId, 10, 64)
	start := codeIdIndex*10000 + 1
	var buffer bytes.Buffer
	buffer.WriteString("{\"code_list\": [")
	for i := int64(0); i < 10000; i++ {
		orderCode := getOrderCode(int64(start + i))
		uniqueCode := createUniqueCode(orderCode, string(secret))
		buffer.WriteString("\"" + uniqueCode + "\"")
		if i < 9999 {
			buffer.WriteString(",")
		}
	}
	buffer.WriteString("]}")

	return fmt.Sprintf("{\"success\": true, \"msg\":\"获取成功\", \"data\": %s}", buffer.String()), nil
}

func checkCode(stub shim.ChaincodeStubInterface, args []string) (string, error) {
	code := args[1]
	orderCode := code[0:10]
	codeId := getCodeId(code)
	value, err := stub.GetState(codeId)
	if value == nil {
		return "{\"success\": false, \"msg\":\"code has not created\"}", nil
	}
	if err != nil {
		return "{\"success\": false, \"msg\":\"get state error\"}", nil
	}

	secret, err := stub.GetPrivateData(PRIVATE_DATA_COLLECTION_NAME, codeId)
	if err != nil {
		return "{\"success\": false, \"msg\":\"get private data error\"}", err
	}
	uniqueCode := createUniqueCode(orderCode, string(secret))
	if code == uniqueCode {
		return "{\"success\": true, \"msg\":\"校验通过\"}", nil
	}
	return "{\"success\": false, \"msg\":\"校验失败\"}", nil
}

func getCreator(stub shim.ChaincodeStubInterface) string {
	// 获取交易提交者的身份（证书中的用户名）
	creatorByte, _ := stub.GetCreator()
	certStart := bytes.IndexAny(creatorByte, "-----BEGIN")
	if certStart == -1 {
		fmt.Errorf("No certificate found")
	}
	certText := creatorByte[certStart:]
	bl, _ := pem.Decode(certText)
	if bl == nil {
		fmt.Errorf("Could not decode the PEM structure")
	}
	cert, err := x509.ParseCertificate(bl.Bytes)
	if err != nil {
		fmt.Errorf("ParseCertificate failed")
	}

	uname := cert.Subject.CommonName
	return uname
}

func getUserId(identityId string) string {
	arr := strings.Split(identityId, "_")
	if len(arr) == 1 {
		return identityId
	} else {
		return arr[1]
	}
}

func intAbs(i int64) int64 {
	if i < 0 {
		return -i
	}
	return i
}

func createSecret(timestamp string, txId string) string {
	param := timestamp + txId
	secret := md5V(param)
	return secret
}

func createUniqueCode(orderCode string, secret string) string {
	tempCode := md5V(orderCode + secret)
	code := orderCode + tempCode[len(tempCode)-6:]
	return code
}

func md5V(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

func getOrderCode(n int64) string {
	orderCode := fmt.Sprintf("%0*d", 10, n)
	return orderCode
}

func getCodeId(uniqueCode string) string {
	orderCode := uniqueCode[0:10]
	codeInt, _ := strconv.ParseInt(orderCode, 10, 64)
	temp := (codeInt - 1) / 10000
	codeId := strconv.FormatInt(temp, 10)
	return codeId
}

func getCurrentCodeId(stub shim.ChaincodeStubInterface) (string, error) {
	value, err := stub.GetState(LAST_CODE_ID)
	if value == nil {
		return "0", nil
	}
	if err != nil {
		return "0", err
	}

	codeInt, _ := strconv.ParseInt(string(value), 10, 64)
	codeId := strconv.FormatInt(codeInt+1, 10)
	return codeId, nil
}

func main() {
	if err := shim.Start(new(UniqueCodeChaincode)); err != nil {
		fmt.Printf("Error starting UniqueCodeChaincode: %s", err)
	}
}
