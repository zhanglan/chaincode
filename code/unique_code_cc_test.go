package main

import (
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"testing"
)

func mockInit(t *testing.T, stub *shim.MockStub, args [][]byte) {
	res := stub.MockInit("1", args)
	if res.Status != shim.OK {
		fmt.Println("Init failed", string(res.Message))
		t.FailNow()
	}
}

func createCodeT(t *testing.T, stub *shim.MockStub, args []string) {
	res := stub.MockInvoke("1", [][]byte{[]byte("invoke"), []byte("createCode"), []byte(args[0])})
	if res.Status != shim.OK {
		fmt.Println("createCodeT failed", string(res.Message))
	}
	payload := string(res.Payload)
	fmt.Println(payload)
}

func listCodeT(t *testing.T, stub *shim.MockStub, args []string) {
	res := stub.MockInvoke("1", [][]byte{[]byte("invoke"), []byte("listCode"), []byte(args[0])})
	if res.Status != shim.OK {
		fmt.Println("listCodeT failed", string(res.Message))
	}
	payload := string(res.Payload)
	fmt.Println(payload)
}

func checkCodeT(t *testing.T, stub *shim.MockStub, args []string) {
	res := stub.MockInvoke("1", [][]byte{[]byte("invoke"), []byte("checkCode"), []byte(args[0])})
	if res.Status != shim.OK {
		fmt.Println("checkCodeT failed", string(res.Message))
	}
	payload := string(res.Payload)
	fmt.Println(payload)
}

func TestUC(t *testing.T) {
	cc := new(UniqueCodeChaincode)
	stub := shim.NewMockStub("UniqueCodeChaincode", cc)
	mockInit(t, stub, nil)
	createCodeT(t, stub, []string{"1576458896541"})
	listCodeT(t, stub, []string{"0"})
	createCodeT(t, stub, []string{"1576458896541"})
	listCodeT(t, stub, []string{"1"})
}
