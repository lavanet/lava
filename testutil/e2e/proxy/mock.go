package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type mockMap struct {
	requests map[string]string //`json:"requests"`
}

func mapToJsonFile(mMap mockMap, outfile string) error {
	// mockMap to json
	jsonData, err := json.Marshal(mMap.requests)
	if err != nil {
		fmt.Println(err)
		return err
	}
	// json to file
	jsonFile, err := os.Create(outfile)
	if err != nil {
		panic(err)
	}
	defer jsonFile.Close()
	jsonFile.Write(jsonData)
	jsonFile.Close()

	fmt.Println("::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::")
	fmt.Println("    JSON data written to ", jsonFile.Name(), " with "+fmt.Sprintf("%d", len(mMap.requests))+" entries")
	fmt.Println("::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::")
	println()

	return nil
}
func jsonFileToMap(jsonfile string) (m map[string]string) {
	// open json file
	m = map[string]string{}
	jsonFile, err := os.Open(jsonfile)
	if err != nil {
		fmt.Println(" ::: XXX ::: Could not open "+jsonfile+" ::: ", err)
		return
	}
	defer jsonFile.Close()

	// Json to mockMap
	byteValue, _ := ioutil.ReadAll(jsonFile)
	if false { // display loaded file contents
		println(" ::::::::::::::::: FILE CONTENTS :::::::::::::::::", jsonFile.Name())
		println(string(byteValue))
		println(" :::::::::::::::::::::::::::::::::::::::::::::::::")
	}
	var newMock mockMap
	json.Unmarshal(byteValue, &newMock.requests)
	count := 0
	for key, body := range newMock.requests {
		// println(" ::: ", key, " ::: ", body)
		jStruct := &jsonStruct{}
		json.Unmarshal([]byte(key), jStruct)
		jStruct.ID = 0
		keyNoId, _ := json.Marshal(jStruct)
		m[string(keyNoId)] = body
		count += 1
	}
	fmt.Println("::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::")
	fmt.Println("    Successfully Loaded " + jsonfile + " with " + fmt.Sprintf("%d", count) + " entries")
	fmt.Println("::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::")
	println()

	return
}
