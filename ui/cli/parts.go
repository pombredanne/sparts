package main

/*
	The functions for the artifact routines can be found in this file.
*/

// Licensing: (Apache-2.0 AND BSD-3-Clause AND BSD-2-Clause)

/*
 * NOTICE:
 * =======
 *  Copyright (c) 2018 Wind River Systems, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at:
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software  distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES
 * OR CONDITIONS OF ANY KIND, either express or implied.
 */

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/tabwriter"
	//"log"
	//"path"
	//"path/filepath"
	"net/http"
	//"os/user"
	"strings"
)

// used in api call to establish relationship between a part and supplier.
type PartSupplierRecord struct {
	PartUUID     string `json:"part_uuid"`     // Part uuid
	SupplierUUID string `json:"supplier_uuid"` // Suppler uuid
}

// used in api call to establish relationship between a part and artifact.
type PartArtifactRecord struct {
	PartUUID     string `json:"part_uuid"`     // Part uuid
	ArtifactUUID string `json:"envelope_uuid"` // Suppler uuid
}

// displayParts displays a list of parts
func displayPartsFromLedger(partsList []PartItemRecord) {
	if len(partsList) == 0 {
		// empty list
		return
	}
	////fmt.Println("  Parts: ")
	for k := range partsList {
		part, err := getPartInfo(partsList[k].PartUUID)
		if err != nil {
			// error retrieving part
			fmt.Println("Could not retrieve part for uuid=", partsList[k].PartUUID)
			continue // skip to next part.
		}
		fmt.Println()
		fmt.Println("    " + _CYAN_FG + part.Name + _COLOR_END)
		fmt.Print("    ")
		fmt.Print()
		///whiteSpace := createWhiteSpace (len (part.Name))
		///fmt.Printf ("%s%s%s\n", createLine (part.Name), "       ", "------------------------" )
		fmt.Println("-------------------------------------------------")

		fmt.Println("    Name: \t " + part.Name)
		fmt.Println("    Version: \t " + part.Version)
		fmt.Println("    UUID: \t " + part.UUID)
		// Format the descriptions greater
		chuckSize := 60
		for len(part.Description) > chuckSize && part.Description[chuckSize] != ' ' {
			chuckSize++
		}
		chuckSize++
		descriptionStr := strings.Join(chunkString(part.Description, chuckSize), "\n                 ")
		fmt.Println("    Description: " + descriptionStr)
		fmt.Println()
	}

}

// displayPartList displays a list of parts.
func displayPartList() {
	var list []PartRecord
	list, err := getPartListFromDBWhere("", "") // "", "" sends back all parts in db
	if checkAndReportError(err) {
		return
	}
	// Let's check if the list of suppliers is empty
	if len(list) == 0 {
		fmt.Printf(" There are no parts cached locally. Try '%s synch ledger'\n", filepath.Base(os.Args[0]))
		return
	}

	//Sort the list
	//list = sortSupplierList(supplierList)

	const padding = 1
	w := tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ',
		tabwriter.Debug)
	//fmt.Fprintf(w, "\n")
	//fmt.Println()
	fmt.Println("    			     PARTS      ")
	//fmt.Printf(" | Supplier: %s\n", "wr")
	fmt.Fprintf(w, "\t%s\t %s\t %s\n", " ------------------", "-----------", "------------------------------------")
	fmt.Fprintf(w, "\t%s\t %s\t %s\n", " Name  ", "   Alias ", "UUID  ")
	fmt.Fprintf(w, "\t%s\t %s\t %s\n", " ------------------", "-----------", "------------------------------------")

	for k := range list {
		alias, _ := getAliasUsingValue(list[k].UUID)
		// format alias field for nil values and for short length ones
		if alias == "" {
			alias = "<none>"
		}
		/*********
		else if len(alias) < 4 {
			alias = "  " + alias
		}
		****/

		fmt.Fprintf(w, "\t %s\t %s\t %s\n", list[k].Name, alias, list[k].UUID)
	}
	//fmt.Println()
	fmt.Fprintf(w, "\n")
	w.Flush()
}

// pushPartToLedger adds part to the ledger
func pushPartToLedger(part PartRecord) error {

	////var part PartRecord
	/****
	part.Name = record.Name
	part.Version = version
	part.Alias = label
	part.Label = label
	part.Licensing = licensing
	//part.URI = url
	part.Description = description
	part.Checksum = checksum
	part.UUID = uuid
	*****************/

	var requestRecord PartAddRecord
	requestRecord.PrivateKey = getLocalConfigValue(_PRIVATE_KEY)
	requestRecord.PublicKey = getLocalConfigValue(_PUBLIC_KEY)
	if requestRecord.PrivateKey == "" || requestRecord.PublicKey == "" {
		return fmt.Errorf("Private and/or Public key(s) are not set \n Use 'sparts config' to set keys")
	}
	requestRecord.Part = part
	var replyRecord ReplyType
	err := sendPostRequest(_PARTS_API, requestRecord, replyRecord)
	if err != nil {
		return err
	}

	return nil
}

// createPartSupplierRelationship establishes a ledger relationship between
// a Part and Supplier.
func createPartSupplierRelationship(part_uuid string, supplier_uuid string) (bool, error) {
	var requestRecord PartToSupplierRecord
	var partSupplierItem PartSupplierPair

	partSupplierItem.PartUUID = part_uuid
	partSupplierItem.SupplierUUID = supplier_uuid

	requestRecord.PrivateKey = getLocalConfigValue(_PRIVATE_KEY)
	requestRecord.PublicKey = getLocalConfigValue(_PUBLIC_KEY)
	if requestRecord.PrivateKey == "" || requestRecord.PublicKey == "" {
		return false, fmt.Errorf("Private and/or Public key(s) are not set \n Use 'sparts config' to set keys")
	}
	requestRecord.Relation = partSupplierItem

	var replyRecord ReplyType
	err := sendPostRequest(_PARTS_TO_SUPPLIER_API, requestRecord, replyRecord)
	if err != nil {
		return false, err
	}
	return true, nil
}

// createPartArtifactRelationship establishes a ledger relationship between
// a Part and Artifact.
func createPartArtifactRelationship(part_uuid string, artifact_uuid string) bool {
	var partArtifactInfo PartArtifactRecord

	partArtifactInfo.PartUUID = part_uuid
	partArtifactInfo.ArtifactUUID = artifact_uuid

	partArtifactAsBytes, err := json.Marshal(partArtifactInfo)
	if err != nil {
		fmt.Printf("Error: %s", err)
		return false
	}

	//fmt.Println (string(supplierAsBytes))
	requestURL := "http://" + getLocalConfigValue(_LEDGER_ADDRESS_KEY) + "/api/sparts/ledger/parts/AddEnvelope"
	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(partArtifactAsBytes))
	if err != nil {
		fmt.Printf("Error: %s", err)
		return false
	}
	req.Header.Set("X-Custom-Header", "PartToArtifact")
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %s", err)
		return false
	}
	defer resp.Body.Close()

	//fmt.Println("response Status:", resp.Status)
	//fmt.Println("response Headers:", resp.Header)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error: %s", err)
		return false
	}
	fmt.Println("PartToArtifact: response Body:", string(body))
	//  {"status":"success"}
	if strings.Contains(string(body), "success") {
		return true
	} else {
		return false
	}
}

// Returns "" if error encountered.
func getRootEnvelope(part PartRecord) string {
	// e.g., "root:3568f20a-8faa-430e-7c65-e9fce9aa155d"
	tokenSize := len(_ROOT_TOKEN) // e.g., _ROOT_TOKEN = "root:"
	if len(part.Label) > tokenSize && part.Label[:tokenSize] == _ROOT_TOKEN {
		// part.Label starts with token characters.
		return part.Label[tokenSize:]
	}
	return ""
}

func getPartInfo(uuid string) (PartRecord, error) {
	var part PartRecord
	////part.Name = ""
	////part.UUID = ""
	//check that uuid is valid.
	if !isValidUUID(uuid) {
		return part, fmt.Errorf("'%s' UUID is not in a valid format", uuid)
	}

	// WORK AROUND - ledger returning wrong format:
	replyAsBytes, err := httpGetAPIRequest(getLocalConfigValue(_LEDGER_ADDRESS_KEY),
		_PARTS_API+"/"+uuid)

	err = json.Unmarshal(replyAsBytes, &part)
	if err != nil {
		if _DEBUG_DISPLAY_ON {
			displayErrorMsg(err.Error())
		}
		if _DEBUG_REST_API_ON {
			fmt.Printf("\n%s\n", replyAsBytes)
		}
		return part, fmt.Errorf("Ledger response may not be properly formatted")
	}

	/*******
	// WORK AROUND - This is what it SHOULD BE
	err := sendGetRequest(getLocalConfigValue(_LEDGER_ADDRESS_KEY), _PARTS_API+"/"+uuid, &part)
	if err != nil {
		// error occurred - return err
		return part, err
	}
	*****/
	/*********
		// TODO: do we need to check returned uuid is same?
		if part.UUID != uuid {
			return part, errors.New(fmt.Sprintf("Part not found in ledger with uuid = '%s'", uuid))
		}
		return part, nil
	}
	*****/
	return part, err
}

// getPartList retrieves the list of all parts from the ledger.
func getPartListFromLedger() ([]PartRecord, error) {
	var partList = []PartRecord{}
	err := sendGetRequest(getLocalConfigValue(_LEDGER_ADDRESS_KEY), _PARTS_API, &partList)
	return partList, err
}
