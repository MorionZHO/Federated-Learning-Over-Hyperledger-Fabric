package API

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"
)

const (
	mspID        = "Org1MSP"
	cryptoPath   = "E:/fabric-test/fabric-samples-main/test-network/organizations/peerOrganizations/org1.example.com"
	certPath     = cryptoPath + "/users/User1@org1.example.com/msp/signcerts/User1@org1.example.com-cert.pem"
	keyPath      = cryptoPath + "/users/User1@org1.example.com/msp/keystore/"
	tlsCertPath  = cryptoPath + "/peers/peer0.org1.example.com/tls/ca.crt"
	peerEndpoint = "localhost:7051"
	gatewayPeer  = "peer0.org1.example.com"
)

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

type ModelParamDy struct {
	Params  map[string]interface{} `json:"params"`
	UserID  string                 `json:"userID"`
	RoundID string                 `json:"roundID"`
}

type ExistGroups struct {
	GroupsName []string `json:"groupsName"`
}

func ResigerUser(groupname string, userId string) error {
	// The gRPC client connection should be shared by all Gateway connections to this endpoint
	clientConnection := newGrpcConnection()

	id := newIdentity()
	sign := newSign()

	// Create a Gateway connection for a specific client identity
	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithClientConnection(clientConnection),
		// Default timeouts for different gRPC calls
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer gw.Close()

	// Override default values for chaincode and channel name as they may differ in testing contexts.
	chaincodeName := "FL"
	if ccname := os.Getenv("CHAINCODE_NAME"); ccname != "" {
		chaincodeName = ccname

	}

	channelName := "mychannel"
	if cname := os.Getenv("CHANNEL_NAME"); cname != "" {
		channelName = cname
	}

	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

	fmt.Printf("\n--> Submit Transaction: RegisterUser \n")

	_, err = contract.SubmitTransaction("RegisterUser", groupname, userId)
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
		return err
	}

	fmt.Printf("*** Transaction committed successfully\n")
	return err
}

// Upload model parameters to fabric
func UploadModelParam(filepath string, groupname string, roundId string, userId string) error {
	var param MyModelParams
	//Open JSON file
	file, err := os.Open(filepath)
	if err != nil {
		fmt.Println("Error opening JSON file:", err)
		return err
	}
	defer file.Close()

	// Parse JSON into map
	// Map[string]interface{} is used because we don’t know the JSON structure in advance
	if err := json.NewDecoder(file).Decode(&param); err != nil {
		fmt.Println("Error decoding JSON:", err)
		return err
	}

	// Convert structure data to JSON string
	paramsJSON, err := json.Marshal(param)
	fmt.Println("the length of data is ", len(paramsJSON))
	if err != nil {
		log.Fatalf("Failed to marshal MyModelParams to JSON: %v", err)
	}

	// The gRPC client connection should be shared by all Gateway connections to this endpoint
	clientConnection := newGrpcConnection()

	id := newIdentity()
	sign := newSign()

	// Create a Gateway connection for a specific client identity
	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithClientConnection(clientConnection),
		// Default timeouts for different gRPC calls
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer gw.Close()

	// Override default values for chaincode and channel name as they may differ in testing contexts.
	chaincodeName := "FL"
	if ccname := os.Getenv("CHAINCODE_NAME"); ccname != "" {
		chaincodeName = ccname

	}

	channelName := "mychannel"
	if cname := os.Getenv("CHANNEL_NAME"); cname != "" {
		channelName = cname
	}

	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

	fmt.Printf("\n--> Submit Transaction: UploadModelParam \n")

	_, err = contract.SubmitTransaction("UploadModelParam", groupname, roundId, userId, string(paramsJSON))
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
		return err
	}

	fmt.Printf("*** Transaction committed successfully\n")
	return err
}

func UploadModelParamDy(filepath string, groupname string, roundId string, userId string) error {
	//Open JSON file
	file, err := os.Open(filepath)
	if err != nil {
		fmt.Println("Error opening JSON file:", err)
		return err
	}
	defer file.Close()

	// Parse JSON into map
	// Parse the JSON into a map
	var params map[string]interface{}
	if err := json.NewDecoder(file).Decode(&params); err != nil {
		fmt.Println("Error decoding JSON:", err)
		return err
	}

	// Convert the map data to a JSON string
	paramsJSON, err := json.Marshal(params)
	fmt.Println("the length of data is ", len(paramsJSON))
	if err != nil {
		log.Fatalf("Failed to marshal params to JSON: %v", err)
	}

	// The gRPC client connection should be shared by all Gateway connections to this endpoint
	clientConnection := newGrpcConnection()

	id := newIdentity()
	sign := newSign()

	// Create a Gateway connection for a specific client identity
	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithClientConnection(clientConnection),
		// Default timeouts for different gRPC calls
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer gw.Close()

	// Override default values for chaincode and channel name as they may differ in testing contexts.
	chaincodeName := "FL"
	if ccname := os.Getenv("CHAINCODE_NAME"); ccname != "" {
		chaincodeName = ccname

	}

	channelName := "mychannel"
	if cname := os.Getenv("CHANNEL_NAME"); cname != "" {
		channelName = cname
	}

	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

	fmt.Printf("\n--> Submit Transaction: UploadModelParam \n")

	_, err = contract.SubmitTransaction("UploadModelParam", groupname, roundId, userId, string(paramsJSON))
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
		return err
	}

	fmt.Printf("*** Transaction committed successfully\n")
	return err
}

// Read model parameters based on roundID
func ReadModelParam(groupname string, roundId string) error {
	// The gRPC client connection should be shared by all Gateway connections to this endpoint
	clientConnection := newGrpcConnection()

	id := newIdentity()
	sign := newSign()

	// Create a Gateway connection for a specific client identity
	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithClientConnection(clientConnection),
		// Default timeouts for different gRPC calls
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer gw.Close()

	// Override default values for chaincode and channel name as they may differ in testing contexts.
	chaincodeName := "FL"
	if ccname := os.Getenv("CHAINCODE_NAME"); ccname != "" {
		chaincodeName = ccname

	}

	channelName := "mychannel"
	if cname := os.Getenv("CHANNEL_NAME"); cname != "" {
		channelName = cname
	}

	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

	fmt.Printf("\n--> Submit Transaction: GetAggregatedParams \n")
	//EvaluateTransaction is Query,SubmitTransaction is Modify
	evaluateResult, err := contract.EvaluateTransaction("GetAggregatedParams", groupname, roundId)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	fmt.Printf("*** Transaction committed successfully\n")
	var param ModelParam
	err = json.Unmarshal(evaluateResult, &param)
	if err != nil {
		log.Fatalf("Error occurred during unmarshalling. Error: %s", err.Error())
	}

	prettyJSON, err := json.MarshalIndent(param.Params, "", "    ")
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return err
	}
	if err = ioutil.WriteFile(groupname+"_AllUser_Round"+param.RoundID+".json", prettyJSON, 0644); err != nil {
		fmt.Println("Error writing output JSON file:", err)
		return err
	}
	return nil
}

func ReadUserModel(key string) error {
	// The gRPC client connection should be shared by all Gateway connections to this endpoint
	clientConnection := newGrpcConnection()

	id := newIdentity()
	sign := newSign()

	// Create a Gateway connection for a specific client identity
	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithClientConnection(clientConnection),
		// Default timeouts for different gRPC calls
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer gw.Close()

	// Override default values for chaincode and channel name as they may differ in testing contexts.
	chaincodeName := "FL"
	if ccname := os.Getenv("CHAINCODE_NAME"); ccname != "" {
		chaincodeName = ccname

	}

	channelName := "mychannel"
	if cname := os.Getenv("CHANNEL_NAME"); cname != "" {
		channelName = cname
	}

	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

	//EvaluateTransaction is Query,SubmitTransaction is Modify
	evaluateResult, err := contract.EvaluateTransaction("GetParam", key)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	var param ModelParam
	err = json.Unmarshal(evaluateResult, &param)
	if err != nil {
		log.Fatalf("Error occurred during unmarshalling. Error: %s", err.Error())
	}

	prettyJSON, err := json.MarshalIndent(param.Params, "", "    ")
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return err
	}
	if err = ioutil.WriteFile(param.UserID+"_"+param.RoundID+".json", prettyJSON, 0644); err != nil {
		fmt.Println("Error writing output JSON file:", err)
		return err
	}
	return nil
}

func ReadUserModel_Dy(key string) error {
	// The gRPC client connection should be shared by all Gateway connections to this endpoint
	clientConnection := newGrpcConnection()

	id := newIdentity()
	sign := newSign()

	// Create a Gateway connection for a specific client identity
	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithClientConnection(clientConnection),
		// Default timeouts for different gRPC calls
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer gw.Close()

	// Override default values for chaincode and channel name as they may differ in testing contexts.
	chaincodeName := "FL"
	if ccname := os.Getenv("CHAINCODE_NAME"); ccname != "" {
		chaincodeName = ccname

	}

	channelName := "mychannel"
	if cname := os.Getenv("CHANNEL_NAME"); cname != "" {
		channelName = cname
	}

	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

	//EvaluateTransaction is Query,SubmitTransaction is Modify
	evaluateResult, err := contract.EvaluateTransaction("GetParam", key)
	if err != nil {
		return fmt.Errorf("failed to evaluate transaction: %v", err)
	}
	// Unmarshal the result into the ModelParam structure
	var modelParam ModelParamDy
	err = json.Unmarshal(evaluateResult, &modelParam)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON data: %v", err)
	}

	// Marshal only the Params field of the ModelParam
	prettyJSON, err := json.MarshalIndent(modelParam.Params, "", "    ")
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return err
	}
	if err = ioutil.WriteFile("./modelData/"+key+"_Dy.json", prettyJSON, 0644); err != nil {
		fmt.Println("Error writing output JSON file:", err)
		return err
	}
	fmt.Println("file successfully save to " + key + "_" + modelParam.UserID + "_" + modelParam.RoundID + "_Dy.json")
	return nil
}

func GetExistGroupNameList() error {
	// The gRPC client connection should be shared by all Gateway connections to this endpoint
	clientConnection := newGrpcConnection()

	id := newIdentity()
	sign := newSign()

	// Create a Gateway connection for a specific client identity
	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithClientConnection(clientConnection),
		// Default timeouts for different gRPC calls
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer gw.Close()

	// Override default values for chaincode and channel name as they may differ in testing contexts.
	chaincodeName := "FL"
	if ccname := os.Getenv("CHAINCODE_NAME"); ccname != "" {
		chaincodeName = ccname

	}

	channelName := "mychannel"
	if cname := os.Getenv("CHANNEL_NAME"); cname != "" {
		channelName = cname
	}

	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

	//EvaluateTransaction is Query,SubmitTransaction is Modify
	evaluateResult, err := contract.EvaluateTransaction("GetGroupsNameList")
	if err != nil {
		return fmt.Errorf("failed to evaluate transaction: %v", err)
	}
	// Unmarshal the result into the ModelParam structure
	var GroupList ExistGroups
	err = json.Unmarshal(evaluateResult, &GroupList)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON data: %v", err)
	}
	fmt.Println(GroupList)
	return nil
}

// newGrpcConnection creates a gRPC connection to the Gateway server.
func newGrpcConnection() *grpc.ClientConn {
	certificate, err := loadCertificate(tlsCertPath)
	if err != nil {
		panic(err)
	}

	//Use the Org1 user's X.509 certificate as the client identity and use the signature implementation based on that user's private key
	certPool := x509.NewCertPool()
	//Add certificate
	certPool.AddCert(certificate)
	transportCredentials := credentials.NewClientTLSFromCert(certPool, gatewayPeer)

	connection, err := grpc.Dial(peerEndpoint, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		panic(fmt.Errorf("failed to create gRPC connection: %w", err))
	}

	return connection
}

// newIdentity creates a client identity for this Gateway connection using an X.509 certificate.
func newIdentity() *identity.X509Identity {
	certificate, err := loadCertificate(certPath)
	if err != nil {
		panic(err)
	}

	id, err := identity.NewX509Identity(mspID, certificate)
	if err != nil {
		panic(err)
	}

	return id
}

func loadCertificate(filename string) (*x509.Certificate, error) {
	certificatePEM, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}
	return identity.CertificateFromPEM(certificatePEM)
}

// newSign creates a function that generates a digital signature from a message digest using a private key.
func newSign() identity.Sign {
	files, err := os.ReadDir(keyPath)
	if err != nil {
		panic(fmt.Errorf("failed to read private key directory: %w", err))
	}
	privateKeyPEM, err := os.ReadFile(path.Join(keyPath, files[0].Name()))

	if err != nil {
		panic(fmt.Errorf("failed to read private key file: %w", err))
	}

	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		panic(err)
	}

	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		panic(err)
	}

	return sign
}

// Format JSON data
func formatJSON(data []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
		panic(fmt.Errorf("failed to parse JSON: %w", err))
	}
	return prettyJSON.String()
}
