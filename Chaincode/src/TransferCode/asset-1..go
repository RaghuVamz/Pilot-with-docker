package main

import (
	"errors"
	"fmt"
	"strconv"
	//"strconv"
	"encoding/json"
	"time"
	//"strings"
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

var assestIndexstr = "_assestindex"

// AssetObject struct
type AssetObject struct {
	Serialno string
	Partno   string
	Owner    string
}

//==============================================================================================================================
//	 Status types - contract lifecycle is broken down into 5 statuses, this is part of the business logic to determine what can
//					be done to the vehicle at points in it's lifecycle
//==============================================================================================================================
const STATE_OPEN = 0
const STATE_READYFORSHIPMENT = 1
const STATE_INTRANSIT = 2
const STATE_SHIPMENT_REACHED = 3
const STATE_SHIPMENT_DELIVERED = 4

const SELLER = "seller"
const TRANSPORTER = "transporter"
const BUYER = "lease_company"

// SalesContractObject struct
type SalesContractObject struct {
	Contractid  string
	Stage       int
	Buyer       string
	Transporter string
	Seller      string
	AssetID     string
	DocumentID  string
	TimeStamp   string // This is the time stamp
}

func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

// Init initializes the chain
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	var err error
	var empty []string
	jsonAsBytes, _ := json.Marshal(empty)
	err = stub.PutState(assestIndexstr, jsonAsBytes)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// Invoke is our entry point to invoke a chaincode function
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "init" {
		return t.Init(stub, "init", args)
	} else if function == "initAssset" {
		return t.initAssset(stub, args)
	} else if function == "ownerUpdation" {
		return t.updateOwner(stub, args)
	} else if function == "initContract" {
		return t.initContract(stub, args)
	} else if function == "contractUpdation" {
		return t.updateContract(stub, args)
	} else if function == "readyForShipment" {
		return t.toReadyForShipment(stub, args)
	} else if function == "inTransit" {
		return t.toInTransit(stub, args)
	} else if function == "shipmentReached" {
		return t.toShipmentReached(stub, args)
	} else if function == "shipmentDelivered" {
		return t.toShipmentDelivered(stub, args)
	}
	fmt.Println("invoke did not find func: " + function) //error

	return nil, errors.New("Received unknown function invocation: " + function)
}

// Query queries the hyperledger
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("query is running " + function)

	// Handle different functions
	if function == "readState" { //read a variable
		return t.readState(stub, args)
	}
	if function == "keys" {
		return t.getAllKeys(stub, args)
	}
	if function == "readContract" { //read a contract
		return t.readContract(stub, args)
	}
	fmt.Println("query did not find func: " + function) //error

	return nil, errors.New("Received unknown function query " + function)
}

func (t *SimpleChaincode) initAssset(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var err error

	//convert the arguments into an asset Object
	AssetObject, err := CreateAssetObject(args[0:])
	if err != nil {
		fmt.Println("initAsset(): Cannot create asset object ")
		return nil, errors.New("initAsset(): Cannot create asset object")
	}

	// check if the asset already exists
	assestAsBytes, err := stub.GetState(AssetObject.Serialno)
	if err != nil {
		fmt.Println("initAssset() : failed to get asset")
		return nil, errors.New("Failed to get asset")
	}
	if assestAsBytes != nil {
		fmt.Println("initAssset() : Asset already exists ", AssetObject.Serialno)
		jsonResp := "{\"Error\":\"Failed - Asset already exists " + AssetObject.Serialno + "\"}"
		return nil, errors.New(jsonResp)
	}

	buff, err := ARtoJSON(AssetObject)
	if err != nil {
		errorStr := "initAssset() : Failed Cannot create object buffer for write : " + args[1]
		fmt.Println(errorStr)
		return nil, errors.New(errorStr)
	} else {
		err = stub.PutState(args[0], buff)
		if err != nil {
			fmt.Println("initAssset() : write error while inserting record\n")
			return nil, errors.New("initAssset() : write error while inserting record : " + err.Error())
		}
	}
	return nil, nil
}

func (t *SimpleChaincode) initContract(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var err error

	//convert the arguments into an asset Object
	contractObject, err := CreateContractObject(args[0:])
	if err != nil {
		fmt.Println("initContract(): Cannot create contract object ")
		return nil, errors.New("initAsset(): Cannot create contract object")
	}

	// check if the contract already exists
	contractAsBytes, err := stub.GetState(contractObject.Contractid)
	if err != nil {
		fmt.Println("initContract() : failed to get contract")
		return nil, errors.New("Failed to get contract")
	}
	if contractAsBytes != nil {
		fmt.Println("initContract() : contract already exists for ", contractObject.Contractid)
		jsonResp := "{\"Error\":\"Failed - contract already exists " + contractObject.Contractid + "\"}"
		return nil, errors.New(jsonResp)
	}

	buff, err := CTRCTtoJSON(contractObject)
	if err != nil {
		errorStr := "initContract() : Failed Cannot create object buffer for write : " + args[1]
		fmt.Println(errorStr)
		return nil, errors.New(errorStr)
	}
	err = stub.PutState(args[0], buff)
	if err != nil {
		fmt.Println("initContract() : write error while inserting record\n")
		return nil, errors.New("initContract() : write error while inserting record : " + err.Error())
	}
	return nil, nil
}

// read function return value
func (t *SimpleChaincode) readState(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var name, jsonResp string
	var err error

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting name of the var to query")
	}

	name = args[0]
	valAsbytes, err := stub.GetState(name)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + name + "\"}"
		return nil, errors.New(jsonResp)
	}

	return valAsbytes, nil
}

// read function return value
func (t *SimpleChaincode) readContract(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var name, jsonResp string
	var err error

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting name of the var to query")
	}

	name = args[0]
	valAsbytes, err := stub.GetState(name)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + name + "\"}"
		return nil, errors.New(jsonResp)
	}

	return valAsbytes, nil
}

// read function return value
func (t *SimpleChaincode) updateOwner(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var jsonResp string
	var err error

	if len(args) != 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 2 args")
	}

	serialNo := args[0]
	newOwner := args[1]
	valAsbytes, err := stub.GetState(serialNo)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + serialNo + "\"}"
		return nil, errors.New(jsonResp)
	}
	dat, err := JSONtoArgs(valAsbytes)
	if err != nil {
		return nil, errors.New("unable to convert jsonToArgs for" + serialNo)
	}
	fmt.Println(dat)

	serialFromLedger := dat["Serialno"].(string)
	fmt.Println(serialFromLedger)
	partFromLeger := dat["Partno"].(string)
	fmt.Println(partFromLeger)

	myAsset := AssetObject{serialFromLedger, partFromLeger, newOwner}

	buff, err := ARtoJSON(myAsset)
	if err != nil {
		errorStr := "initAssset() : Failed Cannot create object buffer for write : " + args[1]
		fmt.Println(errorStr)
		return nil, errors.New(errorStr)
	}
	err = stub.PutState(serialFromLedger, buff)
	if err != nil {
		fmt.Println("initAssset() : write error while inserting record\n")
		return nil, errors.New("initAssset() : write error while inserting record : " + err.Error())
	}
	return nil, nil
}

// read function return value
func (t *SimpleChaincode) updateContract(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var jsonResp string
	var err error

	if len(args) != 3 {
		return nil, errors.New("Incorrect number of arguments. Expecting 3 args")
	}

	Contractid := args[0]
	NewDocumentID := args[1]
	Newstage, err := strconv.Atoi(args[2])
	if err != nil {
		fmt.Println("updateContract(): Stage should be an integer create failed! ")
		return nil, errors.New("updateContract(): Stage should be an integer create failed. ")
	}
	contractAsbytes, err := stub.GetState(Contractid)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + Contractid + "\"}"
		return nil, errors.New(jsonResp)
	}
	dat, err := JSONtoArgs(contractAsbytes)
	if err != nil {
		return nil, errors.New("unable to convert jsonToArgs for" + Contractid)
	}
	fmt.Println(dat)

	updatedContract := SalesContractObject{dat["Contractid"].(string), Newstage, dat["Buyer"].(string), dat["Transporter"].(string), dat["Seller"].(string), dat["AssetID"].(string), NewDocumentID, time.Now().Format("20060102150405")}

	buff, err := CTRCTtoJSON(updatedContract)
	if err != nil {
		errorStr := "updateContract() : Failed Cannot create object buffer for write : " + args[0]
		fmt.Println(errorStr)
		return nil, errors.New(errorStr)
	}
	err = stub.PutState(dat["Contractid"].(string), buff)
	if err != nil {
		fmt.Println("initAssset() : write error while inserting record\n")
		return nil, errors.New("initAssset() : write error while inserting record : " + err.Error())
	}
	return nil, nil
}

func (t *SimpleChaincode) getAllKeys(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	if len(args) < 2 {
		return nil, errors.New("put operation must include two arguments, a key and value")
	}

	startKey := args[0]
	endKey := args[1]

	keysIter, err := stub.RangeQueryState(startKey, endKey)

	if err != nil {
		return nil, errors.New(fmt.Sprintf("keys operation failed. Error accessing state: %s", err))
	}
	defer keysIter.Close()
	var keys []string
	for keysIter.HasNext() {
		response, _, iterErr := keysIter.Next()
		if iterErr != nil {
			return nil, errors.New(fmt.Sprintf("keys operation failed. Error accessing state: %s", err))
		}
		keys = append(keys, response)
	}

	for key, value := range keys {
		fmt.Printf("key %d contains %s\n", key, value)
	}

	jsonKeys, err := json.Marshal(keys)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("keys operation failed. Error accessing state: %s", err))
	}

	return jsonKeys, nil
}

// CreateAssetObject creates an asset
func CreateAssetObject(args []string) (AssetObject, error) {
	// S001 LHTMO bosch
	var err error
	var myAsset AssetObject

	// Check there are 3 Arguments provided as per the the struct
	if len(args) != 3 {
		fmt.Println("CreateAssetObject(): Incorrect number of arguments. Expecting 3 ")
		return myAsset, errors.New("CreateAssetObject(): Incorrect number of arguments. Expecting 3 ")
	}

	// Validate Serialno is an integer

	_, err = strconv.Atoi(args[0])
	if err != nil {
		fmt.Println("CreateAssetObject(): SerialNo should be an integer create failed! ")
		return myAsset, errors.New("CreateAssetbject(): SerialNo should be an integer create failed. ")
	}

	myAsset = AssetObject{args[0], args[1], args[2]}

	fmt.Println("CreateAssetObject(): Asset Object created: ", myAsset.Serialno, myAsset.Partno, myAsset.Owner)
	return myAsset, nil
}

// CreateContractObject creates an contract
func CreateContractObject(args []string) (SalesContractObject, error) {
	// S001 LHTMO bosch
	var err error
	var myContract SalesContractObject

	// Check there are 3 Arguments provided as per the the struct
	if len(args) != 8 {
		fmt.Println("CreateContractObject(): Incorrect number of arguments. Expecting 8 ")
		return myContract, errors.New("CreateContractObject(): Incorrect number of arguments. Expecting 8 ")
	}

	// Validate Serialno is an integer

	stage, err := strconv.Atoi(args[1])
	if err != nil {
		fmt.Println("CreateAssetObject(): Stage should be an integer create failed! ")
		return myContract, errors.New("CreateAssetbject(): Stage should be an integer create failed. ")
	}
	if stage != 0 {
		fmt.Println("CreateAssetObject(): Stage should be set as open ")
		return myContract, errors.New("CreateAssetbject(): Stage should be set as open")
	}

	myContract = SalesContractObject{args[0], STATE_OPEN, args[2], args[3], args[4], args[5], args[6], time.Now().Format("20060102150405")}

	fmt.Println("CreateContractObject(): Contract Object created: ", myContract.Contractid, myContract.Stage, myContract.Buyer, myContract.Transporter, myContract.Seller, myContract.AssetID, myContract.DocumentID, time.Now().Format("20060102150405"))
	return myContract, nil
}

// ARtoJSON Converts an Asset Object to a JSON String
func ARtoJSON(ast AssetObject) ([]byte, error) {

	ajson, err := json.Marshal(ast)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return ajson, nil
}

// CTRCTtoJSON Converts an contract Object to a JSON String
func CTRCTtoJSON(c SalesContractObject) ([]byte, error) {

	cjson, err := json.Marshal(c)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return cjson, nil
}

// JSON To args[] - return a map of the JSON string
func JSONtoArgs(Avalbytes []byte) (map[string]interface{}, error) {

	var data map[string]interface{}

	if err := json.Unmarshal(Avalbytes, &data); err != nil {
		return nil, err
	}

	return data, nil
}

//	 Transfer Functions
//	 seller to transporter

func (t *SimpleChaincode) toReadyForShipment(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	contractid := args[0]
	caller := args[1]
	callerAffiliation := args[2]
	newDocumentID := args[3]
	// check if the contract exists
	sc, err := getContractObject(stub, contractid)
	if err != nil {
		fmt.Println("sellerToTransporter() : failed to get contract object")
		return nil, errors.New("Failed to get contract object")
	}

	if sc.Stage == STATE_OPEN &&
		sc.Seller == caller &&
		callerAffiliation == SELLER {
		sc.Stage = STATE_READYFORSHIPMENT // and mark it in the state of ready for shipment
		sc.DocumentID = newDocumentID     //attach the new document
	} else { // Otherwise if there is an error
		fmt.Printf("sellerToTransporter: Permission Denied")
		return nil, errors.New(fmt.Sprintf("Permission Denied. sellerToTransporter"))

	}

	status, err := t.save_changes(stub, sc) // Write new state
	if err != nil {
		fmt.Printf("sellerToTransporter: Error saving changes: %s", err)
		return nil, errors.New("Error saving changes")
	}
	fmt.Println("sellerToTransporter: Transfer complete : %s", status)
	return nil, nil // We are Done

}

//	 transporter

func (t *SimpleChaincode) toInTransit(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	contractid := args[0]
	caller := args[1]
	callerAffiliation := args[2]
	// check if the contract exists
	sc, err := getContractObject(stub, contractid)
	if err != nil {
		fmt.Println("toInTransit() : failed to get contract object")
		return nil, errors.New("Failed to get contract object")
	}

	if sc.Stage == STATE_READYFORSHIPMENT &&
		sc.Transporter == caller &&
		callerAffiliation == TRANSPORTER {
		sc.Stage = STATE_INTRANSIT // and mark it in the state of ready for shipment
	} else { // Otherwise if there is an error
		fmt.Printf("toInTransit: Permission Denied")
		return nil, errors.New(fmt.Sprintf("Permission Denied. toInTransit"))

	}

	status, err := t.save_changes(stub, sc) // Write new state
	if err != nil {
		fmt.Printf("toInTransit: Error saving changes: %s", err)
		return nil, errors.New("Error saving changes")
	}
	fmt.Println("toInTransit: Transfer complete : %s", status)
	return nil, nil // We are Done

}

//	 shipment reached

func (t *SimpleChaincode) toShipmentReached(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	contractid := args[0]
	caller := args[1]
	callerAffiliation := args[2]
	// check if the contract exists
	sc, err := getContractObject(stub, contractid)
	if err != nil {
		fmt.Println("toShipmentReached() : failed to get contract object")
		return nil, errors.New("Failed to get contract object")
	}

	if sc.Stage == STATE_INTRANSIT &&
		sc.Transporter == caller &&
		callerAffiliation == TRANSPORTER {
		sc.Stage = STATE_SHIPMENT_REACHED // and mark it in the state of ready for shipment
	} else { // Otherwise if there is an error
		fmt.Printf("toShipmentReached() : Permission Denied")
		return nil, errors.New(fmt.Sprintf("Permission Denied. toInTransit"))

	}

	status, err := t.save_changes(stub, sc) // Write new state
	if err != nil {
		fmt.Printf("toShipmentReached() : Error saving changes: %s", err)
		return nil, errors.New("Error saving changes")
	}
	fmt.Println("toShipmentReached() : Transfer complete : %s", status)
	return nil, nil // We are Done

}

//	 shipment reached

func (t *SimpleChaincode) toShipmentDelivered(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	contractid := args[0]
	caller := args[1]
	callerAffiliation := args[2]
	// check if the contract exists
	sc, err := getContractObject(stub, contractid)
	if err != nil {
		fmt.Println("toShipmentDelivered() : failed to get contract object")
		return nil, errors.New("Failed to get contract object")
	}

	if sc.Stage == STATE_SHIPMENT_REACHED &&
		sc.Buyer == caller &&
		callerAffiliation == BUYER {
		sc.Stage = STATE_SHIPMENT_DELIVERED // and mark it in the state of ready for shipment
	} else { // Otherwise if there is an error
		fmt.Printf("toShipmentDelivered() : Permission Denied")
		return nil, errors.New(fmt.Sprintf("Permission Denied. toInTransit"))

	}

	status, err := t.save_changes(stub, sc) // Write new state
	if err != nil {
		fmt.Printf("toShipmentDelivered() : Error saving changes: %s", err)
		return nil, errors.New("Error saving changes")
	}
	fmt.Println("toShipmentDelivered() : Transfer complete : %s", status)
	return nil, nil // We are Done
}

// save_changes - Writes to the ledger the Contract struct passed in a JSON format. Uses the shim file's
//				  method 'PutState'.
func (t *SimpleChaincode) save_changes(stub shim.ChaincodeStubInterface, sc SalesContractObject) (bool, error) {

	bytes, err := json.Marshal(sc)

	if err != nil {
		fmt.Printf("SAVE_CHANGES: Error converting contract : %s", err)
		return false, errors.New("Error converting contract ")
	}

	err = stub.PutState(sc.Contractid, bytes)

	if err != nil {
		fmt.Printf("SAVE_CHANGES: Error storing contract : %s", err)
		return false, errors.New("Error storing contract")
	}
	return true, nil
}

func getContractObject(stub shim.ChaincodeStubInterface, contractID string) (SalesContractObject, error) {

	// check that the contract already exists
	var sco SalesContractObject
	contractAsBytes, err := stub.GetState(contractID)
	if err != nil {
		fmt.Println("getcontractObject() : failed to get contract")
		return sco, errors.New("Failed to get contract")
	}
	if contractAsBytes == nil {
		fmt.Println("getcontractObject() : erreneous contact object for", contractID)
		jsonResp := "{\"Error\":\"Failed - erreneous contact object for" + contractID + "\"}"
		return sco, errors.New(jsonResp)
	}
	dat, err := JSONtoArgs(contractAsBytes)
	if err != nil {
		fmt.Println("getcontractObject() : failed to convert to object")
		return sco, errors.New("Failed to convert to object")
	}
	stage := dat["Stage"].(float64)
	salesContract := SalesContractObject{dat["Contractid"].(string), int(stage), dat["Buyer"].(string), dat["Transporter"].(string), dat["Seller"].(string), dat["AssetID"].(string), dat["DocumentID"].(string), dat["TimeStamp"].(string)}
	return salesContract, nil
}
