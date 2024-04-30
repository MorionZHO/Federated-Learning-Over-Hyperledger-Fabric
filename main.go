package main

import (
	"Capstone_go/API"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"
)

// layer number should be defined by both python and go client
const layernumber = 4
const pythonPath = "E:/anaconda/envs/CapStone/python.exe"
const scriptPathTrain = "E:/CapStone/flower_tutorial1/train.py"
const scriptPathLoadAndTrain = "E:/CapStone/flower_tutorial1/Load_Param_Train.py"

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

func main() {

	//RoundProcess("Astar", "zhh", "zhy", "none", "1")
	TotalProcess("Astar_test2", []string{"zhh", "zhy", "zzh", "sjg", "other"}, 3)

	//Aggrekey := "icbc_AGGREPARAM_1"
	//API.UploadModelParamDy(filePath1, "icbc", "1", "zhh")
	//API.ReadUserModel_Dy(Aggrekey)

}

// maxUser number is 10,depend on flower config
func RoundProcess(groupname string, userlist []string, roundid string, haveRegister bool) {
	if len(userlist) > 10 {
		fmt.Println("user number exceed!")
		return
	}

	//register user
	if !haveRegister {
		TrainProcess(userlist)
		for i := 0; i < len(userlist); i++ {
			err := API.ResigerUser(groupname, userlist[i])
			if err != nil {
				fmt.Println(err)
			}
		}
	}

	//upload user model param
	for i := 0; i < len(userlist); i++ {
		filePath := fmt.Sprintf("./modelData/model_parameters_%d_%dlayer.json", i, layernumber)
		err := API.UploadModelParamDy(filePath, groupname, roundid, userlist[i])
		if err != nil {
			fmt.Println(err)
		}
	}

	//get model param
	for i := 0; i < len(userlist); i++ {
		key := groupname + "_PARAM_" + userlist[i] + "_" + roundid
		err := API.ReadUserModel_Dy(key)
		if err != nil {
			fmt.Println(err)
		}
	}

	Aggrekey := groupname + "_AGGREPARAM_" + roundid
	err := API.ReadUserModel_Dy(Aggrekey)
	if err != nil {
		fmt.Println(err)
	}
	err = API.GetExistGroupNameList()
	if err != nil {
		fmt.Println(err)
	}
}

func TotalProcess(groupname string, userlist []string, roundNum int) {
	for i := 0; i < roundNum; i++ {
		RoundProcess(groupname, userlist, fmt.Sprintf("%d", i), i != 0)
		Aggrekey := groupname + "_AGGREPARAM_" + fmt.Sprintf("%d", i)
		var wg sync.WaitGroup
		//load aggre param , train data and save model param
		for i := 0; i < len(userlist); i++ {
			wg.Add(1)
			go func(i int) {
				fmt.Println("load model params:", fmt.Sprintf("./modelData/"+Aggrekey+"_Dy.json"))
				exePython(pythonPath, scriptPathLoadAndTrain, []string{fmt.Sprintf("./modelData/" + Aggrekey + "_Dy.json"), fmt.Sprintf("%d", i)})
				wg.Done()
			}(i)
		}
		wg.Wait()

	}
}

func TrainProcess(userlist []string) {
	var wg sync.WaitGroup
	//train data and save model param
	for i := 0; i < len(userlist); i++ {
		wg.Add(1)
		go func(i int) {
			arg := []string{fmt.Sprintf("%d", i)}
			exePython(pythonPath, scriptPathTrain, arg)
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func exePython(pythonpath string, scriptpath string, args []string) {

	// Build command, the python3 here may need to be adjusted to python or python3 according to the actual environment
	cmd := exec.Command(pythonpath, append([]string{scriptpath}, args...)...)

	// Run the command and get the output
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error executing deep learning task: %s\n", err)
		return
	}

	// Print the output of a Python script
	fmt.Printf("Output from Python: %s\n", string(output))
}

func LoadJson(filepath string) {
	// 打开JSON文件
	file, err := os.Open(filepath)
	if err != nil {
		fmt.Println("Error opening JSON file:", err)
		return
	}
	defer file.Close()

	// Parse JSON into map
	// Map[string]interface{} is used because we don’t know the JSON structure in advance
	var params MyModelParams
	if err := json.NewDecoder(file).Decode(&params); err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}

	// Write JSON data to output file
	prettyJSON, err := json.MarshalIndent(params, "", "    ")
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}

	if err := ioutil.WriteFile("output.json", prettyJSON, 0644); err != nil {
		fmt.Println("Error writing output JSON file:", err)
		return
	}

	//for k, v := range params.Fc1Bias {
	//	fmt.Println(k)
	//	fmt.Println(v)
	//}
	//exePython(pythonPath, scriptPathLoad, []string{"E:/Capstone_go/model_parameters_1.json"})
	//exePython(pythonPath, scriptPathLoad, []string{"E:/Capstone_go/output.json"})
}
