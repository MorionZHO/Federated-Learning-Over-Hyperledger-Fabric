package main

import (
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"log"
	"math"
)

// SmartContract provides functions
type SmartContract struct {
	contractapi.Contract
}

type MyModelParams struct {
	Conv1Bias   []float64       `json:"conv1.bias"`
	Conv1Weight [][][][]float64 `json:"conv1.weight"`
	Conv2Bias   []float64       `json:"conv2.bias"`
	Conv2Weight [][][][]float64 `json:"conv2.weight"`
	Fc1Bias     []float64       `json:"fc1.bias"`
	Fc1Weight   [][]float64     `json:"fc1.weight"`
	Fc2Bias     []float64       `json:"fc2.bias"`
	Fc2Weight   [][]float64     `json:"fc2.weight"`
	Fc3Bias     []float64       `json:"fc3.bias"`
	Fc3Weight   [][]float64     `json:"fc3.weight"`
}

type ModelParam struct {
	Params  MyModelParams `json:"params"`
	UserID  string        `json:"userID"`
	RoundID string        `json:"roundID"`
}

type Group struct {
	Users []string `json:"users"`
}

func (s *SmartContract) RegisterUser(ctx contractapi.TransactionContextInterface, groupname string, userID string) error {
	data, err := ctx.GetStub().GetState(groupname)
	if err != nil {
		return fmt.Errorf("failed to get the group: %s", err.Error())
	}
	var group Group
	if data != nil {
		// 解析group的状态信息
		err = json.Unmarshal(data, &group)
		if err != nil {
			return fmt.Errorf("failed to unmarshal group: %s", err.Error())
		}
	}

	// 检查用户是否已在组内
	for _, user := range group.Users {
		if user == userID {
			// 用户已注册
			return fmt.Errorf("user %s is already registered in group %s", userID, groupname)
		}
	}
	// 将新用户添加到组中
	group.Users = append(group.Users, userID)

	// 序列化group对象并更新状态
	data, err = json.Marshal(group)
	if err != nil {
		return fmt.Errorf("failed to marshal group: %s", err.Error())
	}

	// 更新链上状态
	err = ctx.GetStub().PutState(groupname, data)
	if err != nil {
		return fmt.Errorf("failed to update group state: %s", err.Error())
	}

	return nil
}

// 上传模型参数，每次上传完后都会调用check函数，检查是否全部用户已经上传，key是"PARAM_"+userID+"_"+roundID
func (s *SmartContract) UploadModelParam(ctx contractapi.TransactionContextInterface, groupname string, roundID string, userID string, paramJson string) error {

	//检查用户是否已经注册
	data, err := ctx.GetStub().GetState(groupname)
	if err != nil {
		return fmt.Errorf("failed to get the group: %s", err.Error())
	}
	var group Group
	if data != nil {
		// 解析group的状态信息
		err = json.Unmarshal(data, &group)
		if err != nil {
			return fmt.Errorf("failed to unmarshal group: %s", err.Error())
		}
	}

	// 检查用户是否已在组内
	flag := 0
	for _, user := range group.Users {
		if user == userID {
			flag = 1
			break
		}
	}

	if flag == 0 {
		return fmt.Errorf("The userID - %s is not registered", userID)
	}

	var params MyModelParams
	err = json.Unmarshal([]byte(paramJson), &params)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	param := ModelParam{
		Params:  params,
		UserID:  userID,
		RoundID: roundID,
	}

	paramJSON, err := json.Marshal(param)
	if err != nil {
		return err
	}
	//key - value
	err = ctx.GetStub().PutState(groupname+"_PARAM_"+userID+"_"+roundID, paramJSON)
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

// 获取聚合后的模型，key是"AGGREPARAM_" + roundID
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

// CheckAllUploaded 检查是否所有用户都已上传参数，如果全部上传则调用聚合函数，具体实现就是维护一个全局的userid数组，遍历该数组在fabric上进行key查询，如果有一个没查到就说明没有上传完全
func (s *SmartContract) checkAllUploaded(ctx contractapi.TransactionContextInterface, groupname string, usersId []string, roundID string) error {
	// 这里应该实现检查逻辑，如果所有用户都上传了，则调用参数聚合方法

	//四舍五入
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

	// 假设大部分用户都已上传，调用参数聚合方法
	s.aggregateParams(ctx, groupname, usersId, roundID)

	return nil
}

// 模型聚合
func (s *SmartContract) aggregateParams(ctx contractapi.TransactionContextInterface, groupname string, usersId []string, roundID string) error {
	userCount := 0
	var aggreParams MyModelParams
	var initialized bool // 标记是否已初始化aggreParams

	for _, v := range usersId {
		key := groupname + "_PARAM_" + v + "_" + roundID
		data, err := ctx.GetStub().GetState(key)
		if err != nil {
			return err
		}
		if data == nil {
			continue // 如果找不到用户参数，跳过此用户
		}

		userCount++
		var params ModelParam
		err = json.Unmarshal(data, &params)
		if err != nil {
			return fmt.Errorf("failed to unmarshal JSON data for key %s: %v", key, err)
		}

		if !initialized {
			aggreParams = params.Params // 使用第一个找到的用户参数初始化aggreParams
			initialized = true
		} else {
			addValues(&aggreParams, &params.Params) // 累加后续用户的参数
		}
	}

	if initialized {
		divideValues(&aggreParams, userCount) // 计算平均值
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
	// 保存聚合后的参数
	return ctx.GetStub().PutState(groupname+"_AGGREPARAM_"+roundID, paramJSON)
}

func addValues(total *MyModelParams, params *MyModelParams) {
	// 这个函数用于将每个位置对应的值相加到总和的结构体中
	// 你需要根据你的数据结构进行适当的调整

	// 示例：Conv1Bias
	for i, value := range params.Conv1Bias {
		total.Conv1Bias[i] += value
	}

	// Conv1Weight
	for i, layer1 := range params.Conv1Weight {
		for j, layer2 := range layer1 {
			for k, layer3 := range layer2 {
				for l, value := range layer3 {
					total.Conv1Weight[i][j][k][l] += value
				}
			}
		}
	}

	// Conv2Bias
	for i, value := range params.Conv2Bias {
		total.Conv2Bias[i] += value
	}

	// Conv2Weight
	for i, layer1 := range params.Conv2Weight {
		for j, layer2 := range layer1 {
			for k, layer3 := range layer2 {
				for l, value := range layer3 {
					total.Conv2Weight[i][j][k][l] += value
				}
			}
		}
	}

	// 示例：Fc1Bias
	for i, value := range params.Fc1Bias {
		total.Fc1Bias[i] += value
	}

	// 示例：Fc1Weight
	for i, slice1 := range params.Fc1Weight {
		for j, value := range slice1 {
			total.Fc1Weight[i][j] += value
		}
	}
	// 示例：Fc2Bias
	for i, value := range params.Fc2Bias {
		total.Fc2Bias[i] += value
	}

	// 示例：Fc2Weight
	for i, slice1 := range params.Fc2Weight {
		for j, value := range slice1 {
			total.Fc2Weight[i][j] += value
		}
	}
	// 示例：Fc3Bias
	for i, value := range params.Fc3Bias {
		total.Fc3Bias[i] += value
	}

	// 示例：Fc3Weight
	for i, slice1 := range params.Fc3Weight {
		for j, value := range slice1 {
			total.Fc3Weight[i][j] += value
		}
	}
}

func divideValues(params *MyModelParams, num int) {
	// 这个函数用于将每个位置对应的值相加到总和的结构体中
	// 你需要根据你的数据结构进行适当的调整

	// Conv1Bias
	if len(params.Conv1Bias) > 0 {
		for i := range params.Conv1Bias {
			params.Conv1Bias[i] /= float64(num)
		}
	}

	// Conv1Weight
	for i, layer1 := range params.Conv1Weight {
		for j, layer2 := range layer1 {
			for k, layer3 := range layer2 {
				for l := range layer3 {
					params.Conv1Weight[i][j][k][l] /= float64(num)
				}
			}
		}
	}

	// Conv2Bias
	if len(params.Conv2Bias) > 0 {
		for i := range params.Conv2Bias {
			params.Conv2Bias[i] /= float64(num)
		}
	}

	// Conv2Weight
	for i, layer1 := range params.Conv2Weight {
		for j, layer2 := range layer1 {
			for k, layer3 := range layer2 {
				for l := range layer3 {
					params.Conv2Weight[i][j][k][l] /= float64(num)
				}
			}
		}
	}

	// 示例：Fc1Bias
	for i := range params.Fc1Bias {
		params.Fc1Bias[i] /= float64(num)
	}

	// 示例：Fc1Weight
	for i, slice1 := range params.Fc1Weight {
		for j := range slice1 {
			params.Fc1Weight[i][j] /= float64(num)
		}
	}
	// 示例：Fc2Bias
	for i := range params.Fc2Bias {
		params.Fc2Bias[i] /= float64(num)
	}

	// 示例：Fc2Weight
	for i, slice1 := range params.Fc2Weight {
		for j := range slice1 {
			params.Fc2Weight[i][j] /= float64(num)
		}
	}
	// 示例：Fc3Bias
	for i := range params.Fc3Bias {
		params.Fc3Bias[i] /= float64(num)
	}

	// 示例：Fc3Weight
	for i, slice1 := range params.Fc3Weight {
		for j := range slice1 {
			params.Fc3Weight[i][j] /= float64(num)
		}
	}
}
func main() {
	orderChaincode, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		log.Panicf("Error creating FL-basic chaincode: %v", err)
	}
	if err := orderChaincode.Start(); err != nil {
		log.Panicf("Error starting FL-basic chaincode: %v", err)
	}
}
