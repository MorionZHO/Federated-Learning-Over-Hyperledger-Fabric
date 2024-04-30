package main

import (
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"log"
	"math"
)

const GroupsNameListKey = "AllGroups"

// SmartContract provides functions for managing an model
type SmartContract struct {
	contractapi.Contract
}

// ModelParam represents a model parameter which can be uploaded by a user
type ModelParam struct {
	Params  map[string]interface{} `json:"params"`
	UserID  string                 `json:"userID"`
	RoundID string                 `json:"roundID"`
}

// Group represents a group of users
type Group struct {
	Users []string `json:"users"`
}

type ExistGroups struct {
	GroupsName []string `json:"groupsName"`
}

// RegisterUser adds a new user to a group
func (s *SmartContract) RegisterUser(ctx contractapi.TransactionContextInterface, groupname string, userID string) error {
	data, err := ctx.GetStub().GetState(groupname)
	if err != nil {
		return fmt.Errorf("failed to get the group: %s", err.Error())
	}

	var group Group
	if data != nil {
		err = json.Unmarshal(data, &group)
		if err != nil {
			return fmt.Errorf("failed to unmarshal group: %s", err.Error())
		}
	} else {
		//get groupsName list
		groupNamesData, err := ctx.GetStub().GetState(GroupsNameListKey)
		if err != nil {
			return fmt.Errorf("failed to get the group: %s", err.Error())
		}

		var existGroups ExistGroups
		//if groupsName list exist,bind json
		if groupNamesData != nil {
			err = json.Unmarshal(groupNamesData, &existGroups)
			if err != nil {
				return fmt.Errorf("failed to unmarshal group: %s", err.Error())
			}
		}
		//if groupName list not exist,new a struct ,append groupName
		existGroups.GroupsName = append(existGroups.GroupsName, groupname)

		//Marshal struct and put state to fabric
		groupNamesData, err = json.Marshal(existGroups)
		if err != nil {
			return fmt.Errorf("failed to marshal group: %s", err.Error())
		}

		err = ctx.GetStub().PutState(GroupsNameListKey, groupNamesData)
		if err != nil {
			return fmt.Errorf("failed to update group state: %s", err.Error())
		}
	}

	for _, user := range group.Users {
		if user == userID {
			return fmt.Errorf("user %s is already registered in group %s", userID, groupname)
		}
	}

	group.Users = append(group.Users, userID)
	data, err = json.Marshal(group)
	if err != nil {
		return fmt.Errorf("failed to marshal group: %s", err.Error())
	}
	err = ctx.GetStub().PutState(groupname, data)
	if err != nil {
		return fmt.Errorf("failed to update group state: %s", err.Error())
	}
	return nil
}

// UploadModelParam allows a user to upload their model parameters
// paramKey = "groupname_PARAM_userID_roundID"
func (s *SmartContract) UploadModelParam(ctx contractapi.TransactionContextInterface, groupname string, roundID string, userID string, paramJson string) error {
	groupData, err := ctx.GetStub().GetState(groupname)
	if err != nil {
		return fmt.Errorf("failed to get the group: %s", err.Error())
	}
	if groupData == nil {
		return fmt.Errorf("group %s does not exist", groupname)
	}

	var group Group
	err = json.Unmarshal(groupData, &group)
	if err != nil {
		return fmt.Errorf("failed to unmarshal group: %s", err.Error())
	}

	found := false
	for _, user := range group.Users {
		if user == userID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("user %s is not registered in group %s", userID, groupname)
	}

	var params map[string]interface{}
	err = json.Unmarshal([]byte(paramJson), &params)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON params: %s", err.Error())
	}

	param := ModelParam{
		Params:  params,
		UserID:  userID,
		RoundID: roundID,
	}
	paramJSON, err := json.Marshal(param)
	if err != nil {
		return fmt.Errorf("failed to marshal ModelParam: %s", err.Error())
	}

	// Creating a unique key for the user's parameters
	paramKey := fmt.Sprintf("%s_PARAM_%s_%s", groupname, userID, roundID)
	err = ctx.GetStub().PutState(paramKey, paramJSON)
	if err != nil {
		return err
	}
	// check if most users upload the params
	return s.checkAllUploaded(ctx, groupname, group.Users, roundID)
}
func (s *SmartContract) GetParam(ctx contractapi.TransactionContextInterface, key string) (*ModelParam, error) {
	paramJSON, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, err
	}
	if paramJSON == nil {
		return nil, fmt.Errorf("the Model Params %s does not exist", key)
	}

	var Modelparam ModelParam
	err = json.Unmarshal(paramJSON, &Modelparam)
	if err != nil {
		return nil, err
	}
	return &Modelparam, nil
}

// Get the aggregated model, the key is "groupname_AGGREPARAM_" + roundID
func (s *SmartContract) GetAggregatedParams(ctx contractapi.TransactionContextInterface, groupname string, roundID string) (*ModelParam, error) {
	key := groupname + "_AGGREPARAM_" + roundID
	paramJSON, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, err
	}
	if paramJSON == nil {
		return nil, fmt.Errorf("the Model Params %s does not exist", key)
	}

	var Modelparam ModelParam
	err = json.Unmarshal(paramJSON, &Modelparam)
	if err != nil {
		return nil, err
	}
	return &Modelparam, nil
}

// Get groups name list
func (s *SmartContract) GetGroupsNameList(ctx contractapi.TransactionContextInterface) (*ExistGroups, error) {
	paramJSON, err := ctx.GetStub().GetState(GroupsNameListKey)
	if err != nil {
		return nil, err
	}
	if paramJSON == nil {
		return nil, fmt.Errorf("the Model Params %s does not exist", GroupsNameListKey)
	}

	var groupsNameList ExistGroups
	err = json.Unmarshal(paramJSON, &groupsNameList)
	if err != nil {
		return nil, err
	}
	return &groupsNameList, nil
}

// CheckAllUploaded checks whether all users have uploaded parameters. If all have been uploaded, the aggregation function is called. The specific implementation is to maintain a global userid array, traverse the array and perform key queries on the fabric. If one is not found, it means that the upload is not complete.
func (s *SmartContract) checkAllUploaded(ctx contractapi.TransactionContextInterface, groupname string, usersId []string, roundID string) error {
	// Checking logic should be implemented here. If all users have uploaded, the parameter aggregation method is called.

	//rounding
	ratio := int(math.Round(0.8 * float64(len(usersId))))
	for i := 0; i < len(usersId); i++ {
		key := groupname + "_PARAM_" + usersId[i] + "_" + roundID
		data, err := ctx.GetStub().GetState(key)
		if err != nil {
			return err
		}
		if data != nil {
			ratio--
		}
	}
	if ratio > 0 {
		return nil
	}

	// Assuming that most users(80%) have uploaded, call the parameter aggregation method
	s.aggregateParams(ctx, groupname, usersId, roundID)

	return nil
}

func (s *SmartContract) aggregateParams(ctx contractapi.TransactionContextInterface, groupname string, usersId []string, roundID string) error {
	userCount := 0
	aggreParams := make(map[string]interface{})
	var initialized bool

	for _, v := range usersId {
		key := groupname + "_PARAM_" + v + "_" + roundID
		data, err := ctx.GetStub().GetState(key)
		if err != nil {
			return err
		}
		if data == nil {
			continue
		}

		userCount++
		var params ModelParam
		err = json.Unmarshal(data, &params)
		if err != nil {
			return fmt.Errorf("failed to unmarshal JSON data for key %s: %v", key, err)
		}

		if !initialized {
			aggreParams = params.Params
			initialized = true
		} else {
			addValues(aggreParams, params.Params)
		}
	}

	if initialized {
		divideValues(aggreParams, userCount)
	} else {
		return fmt.Errorf("no user params found for aggregation")
	}

	param := ModelParam{
		Params:  aggreParams,
		UserID:  "ALL",
		RoundID: roundID,
	}
	paramJSON, err := json.Marshal(param)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(groupname+"_AGGREPARAM_"+roundID, paramJSON)
}

func addValues(total, params map[string]interface{}) {
	for key, value := range params {
		addRecursive(total, key, value)
	}
}

func addRecursive(total map[string]interface{}, key string, newValue interface{}) {
	if existingValue, exists := total[key]; exists {
		// Check type of newValue and call appropriate function
		switch newTypedValue := newValue.(type) {
		case float64:
			total[key] = existingValue.(float64) + newTypedValue
		case []interface{}:
			if existingSlice, ok := existingValue.([]interface{}); ok {
				total[key] = addSlices(existingSlice, newTypedValue)
			}
		default:
			// handle error or unsupported types
		}
	} else {
		// If the key does not exist, just set it
		total[key] = newValue
	}
}

func addSlices(existingSlice, newSlice []interface{}) []interface{} {
	minLength := min(len(existingSlice), len(newSlice))
	resultSlice := make([]interface{}, minLength)
	for i := 0; i < minLength; i++ {
		resultSlice[i] = addDynamic(existingSlice[i], newSlice[i])
	}
	return resultSlice
}

func addDynamic(a, b interface{}) interface{} {
	switch typedA := a.(type) {
	case float64:
		return typedA + b.(float64)
	case []interface{}:
		return addSlices(typedA, b.([]interface{}))
	default:
		// handle error or unsupported types
		return nil
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func divideValues(params map[string]interface{}, num int) {
	for key, value := range params {
		params[key] = divideRecursive(value, num)
	}
}

func divideRecursive(value interface{}, num int) interface{} {
	switch typedValue := value.(type) {
	case float64:
		return typedValue / float64(num)
	case []interface{}:
		for i, v := range typedValue {
			typedValue[i] = divideRecursive(v, num)
		}
		return typedValue
	default:
		// handle error or unsupported types
		return nil
	}
}

func main() {
	chaincode, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		log.Panicf("Error creating model-params chaincode: %v", err)
	}

	if err := chaincode.Start(); err != nil {
		log.Panicf("Error starting model-params chaincode: %v", err)
	}
}
