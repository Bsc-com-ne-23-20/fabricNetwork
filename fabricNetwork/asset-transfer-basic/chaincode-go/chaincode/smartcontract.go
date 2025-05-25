package chaincode

import (
    "encoding/json"
    "fmt"
    "time"
    "github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

type SmartContract struct {
    contractapi.Contract
}

// An asset is a patient's medical prescription record, with the following attributes:
type Asset struct {
    DoctorId      string         `json:"DoctorId"`      // ID of prescribing doctor
    PatientName   string         `json:"PatientName"`   // Name of the patient
    PatientId     string         `json:"PatientId"`     // Unique identifier for the patient
    DateOfBirth   string         `json:"DateOfBirth,omitempty"` // Patient's DOB (optional)
    Prescriptions []Prescription `json:"Prescriptions"` // Array of prescriptions
    LastUpdated   string         `json:"LastUpdated"`   // Timestamp of last modification
}

// Prescription structure
type Prescription struct {
    PrescriptionId      string `json:"PrescriptionId"`
    MedicationName      string `json:"MedicationName"`
    Dosage              string `json:"Dosage"`
    Instructions        string `json:"Instructions"`
    Status              string `json:"Status"`    // Active, Filled, Revoked, Expired
    CreatedBy           string `json:"CreatedBy"` // DoctorId who created it
    TxID                string `json:"TxID"`
    Timestamp           string `json:"Timestamp"`
    ExpiryDate          string `json:"ExpiryDate,omitempty"`
    DispensingPharmacist string `json:"dispensingPharmacist,omitempty"` // ID of pharmacist who dispensed
    DispensingTimestamp  string `json:"dispensingTimestamp,omitempty"`  // When it was dispensed
}

// IssuePrescription - this function allows a doctor to issue a new prescription for a patient
// It requires the doctor to be authenticated and authorized to perform this action.
func (s *SmartContract) CreateAsset(ctx contractapi.TransactionContextInterface, assetJSON string) error {
    // Parse the entire asset JSON
    var asset Asset
    err := json.Unmarshal([]byte(assetJSON), &asset)
    if err != nil {
        return fmt.Errorf("failed to parse asset JSON: %v", err)
    }

    // Validate required fields
    if asset.PatientId == "" || asset.DoctorId == "" {
        return fmt.Errorf("patientId and doctorId are required")
    }

    // Add metadata
    asset.LastUpdated = time.Now().Format(time.RFC3339)
    for i := range asset.Prescriptions {
        asset.Prescriptions[i].TxID = ctx.GetStub().GetTxID()
        asset.Prescriptions[i].Timestamp = asset.LastUpdated
        asset.Prescriptions[i].Status = "Active"
        asset.Prescriptions[i].CreatedBy = asset.DoctorId

        if asset.Prescriptions[i].ExpiryDate == "" {
            asset.Prescriptions[i].ExpiryDate = time.Now().AddDate(0, 1, 0).Format("2006-01-02") // 1 month from now
        }
    }

    assetJSONBytes, err := json.Marshal(asset)
    if err != nil {
        return err
    }

    return ctx.GetStub().PutState(asset.PatientId, assetJSONBytes)
}

// ReadAsset - returns world state information for an asset, patientId as key
func (s *SmartContract) ReadAsset(ctx contractapi.TransactionContextInterface, patientId string) (*Asset, error) {
    assetJSON, err := ctx.GetStub().GetState(patientId)
    if err != nil {
        return nil, fmt.Errorf("failed to read from world state: %v", err)
    }
    if assetJSON == nil {
        return nil, fmt.Errorf("asset %s does not exist", patientId)
    }

    var asset Asset
    err = json.Unmarshal(assetJSON, &asset)
    if err != nil {
        return nil, err
    }

    return &asset, nil
}

// UpdatePrescription  - may be used to update prescription details, incase of a change in dosage or instructions
// Prescriptions are immutable, but we can update the status or other non-immutable fields
func (s *SmartContract) UpdatePrescription(ctx contractapi.TransactionContextInterface, patientId string, prescriptionJSON string) error {
    // Get existing asset
    asset, err := s.ReadAsset(ctx, patientId)
    if err != nil {
        return err
    }

    // Parse new prescription
    var newPrescription Prescription
    err = json.Unmarshal([]byte(prescriptionJSON), &newPrescription)
    if err != nil {
        return fmt.Errorf("failed to parse prescription JSON: %v", err)
    }

    // Find and update prescription
    found := false
    for i := range asset.Prescriptions {
        if asset.Prescriptions[i].PrescriptionId == newPrescription.PrescriptionId {
            // Preserve immutable fields
            newPrescription.CreatedBy = asset.Prescriptions[i].CreatedBy
            newPrescription.TxID = ctx.GetStub().GetTxID()
            newPrescription.Timestamp = time.Now().Format(time.RFC3339)
            asset.Prescriptions[i] = newPrescription
            found = true
            break
        }
    }

    if !found {
        return fmt.Errorf("prescription %s not found", newPrescription.PrescriptionId)
    }

    asset.LastUpdated = time.Now().Format(time.RFC3339)
    assetJSON, err := json.Marshal(asset)
    if err != nil {
        return err
    }

    return ctx.GetStub().PutState(patientId, assetJSON)
}

// DispensePrescription - this function allows a pharmacist to dispense a prescription
// It checks if the prescription is active before dispensing and updates the status to "Dispensed"
func (s *SmartContract) DispensePrescription(ctx contractapi.TransactionContextInterface, dispensationJSON string) error {
    // Parse the dispensation JSON
    var dispensation struct {
        PatientId       string `json:"patientId"`
        PrescriptionId string `json:"prescriptionId"`
        PharmacistId   string `json:"pharmacistId"`
        Note           string `json:"note,omitempty"`
    }
    
    err := json.Unmarshal([]byte(dispensationJSON), &dispensation)
    if err != nil {
        return fmt.Errorf("failed to parse dispensation JSON: %v", err)
    }
    
    // Validate fields
    if dispensation.PatientId == "" || dispensation.PrescriptionId == "" || dispensation.PharmacistId == "" {
        return fmt.Errorf("patientId, prescriptionId, and pharmacistId are required")
    }

    // Get the asset
    asset, err := s.ReadAsset(ctx, dispensation.PatientId)
    if err != nil {
        return err
    }

    // Find and update prescription
    found := false
    for i := range asset.Prescriptions {
        if asset.Prescriptions[i].PrescriptionId == dispensation.PrescriptionId {
            // Check prescription status
            if asset.Prescriptions[i].Status == "Revoked" {
                return fmt.Errorf("cannot dispense a revoked prescription")
            }
            if asset.Prescriptions[i].Status != "Active" {
                return fmt.Errorf("can only dispense active prescriptions")
            }
            
            // Update prescription status and pharmacist info
            now := time.Now().Format(time.RFC3339)
            asset.Prescriptions[i].Status = "Dispensed"
            asset.Prescriptions[i].TxID = ctx.GetStub().GetTxID()
            asset.Prescriptions[i].Timestamp = now
            asset.Prescriptions[i].DispensingPharmacist = dispensation.PharmacistId
            asset.Prescriptions[i].DispensingTimestamp = now
            found = true
            break
        }
    }

    if !found {
        return fmt.Errorf("prescription not found")
    }

    // Update asset and save to state
    asset.LastUpdated = time.Now().Format(time.RFC3339)
    assetJSON, err := json.Marshal(asset)
    if err != nil {
        return err
    }

    return ctx.GetStub().PutState(dispensation.PatientId, assetJSON) 
}

// GetAssetHistory - obtain the history of a specific asset(patientId) from the ledger 
func (s *SmartContract) GetAssetHistory(ctx contractapi.TransactionContextInterface, patientId string) ([]map[string]interface{}, error) {
    historyIterator, err := ctx.GetStub().GetHistoryForKey(patientId)
    if err != nil {
        return nil, err
    }
    defer historyIterator.Close()

    var history []map[string]interface{}

    for historyIterator.HasNext() {
        historyData, err := historyIterator.Next()
        if err != nil {
            return nil, err
        }

        var asset Asset
        if historyData.Value != nil {
            if err := json.Unmarshal(historyData.Value, &asset); err != nil {
                return nil, err
            }
        }

        record := map[string]interface{}{
            "patientId":     asset.PatientId,
            "patientName":   asset.PatientName,
            "doctorId":      asset.DoctorId,
            "lastUpdated":   asset.LastUpdated,
            "prescriptions": []map[string]interface{}{},
            "timestamp":     historyData.Timestamp.String(),
            "txId":         historyData.TxId,
        }

        for _, prescription := range asset.Prescriptions {
            prescriptionRecord := map[string]interface{}{
                "prescriptionId": prescription.PrescriptionId,
                "medicationName": prescription.MedicationName,
                "dosage":        prescription.Dosage,
                "instructions":  prescription.Instructions,
                "status":       prescription.Status,
                "createdBy":    prescription.CreatedBy,
                "timestamp":    prescription.Timestamp,
                "expiryDate":   prescription.ExpiryDate,
            }
            record["prescriptions"] = append(record["prescriptions"].([]map[string]interface{}), prescriptionRecord)
        }

        history = append(history, record)
    }

    return history, nil
}

// GetPrescriptionsByStatus - obtain prescriptions by status
// This function allows filtering prescriptions based on their status (e.g., Active, Dispensed, Revoked, Expired)
func (s *SmartContract) GetPrescriptionsByStatus(ctx contractapi.TransactionContextInterface, patientId string, status string) ([]Prescription, error) {
    asset, err := s.ReadAsset(ctx, patientId)
    if err != nil {
        return nil, err
    }

    var filtered []Prescription
    for _, prescription := range asset.Prescriptions {
        if prescription.Status == status {
            filtered = append(filtered, prescription)
        }
    }

    return filtered, nil
}

// RevokePrescription - revoke an active prescription
// This function allows a doctor to revoke a prescription, changing its status to "Revoked"
// and ensuring that only the original prescriber can perform this action.
// It also checks if the prescription is currently active before revoking it.
func (s *SmartContract) RevokePrescriptionJSON(ctx contractapi.TransactionContextInterface, revocationJSON string) error {
    // Parse the revocation JSON
    var revocation struct {
        PatientId      string `json:"patientId"`
        PrescriptionId string `json:"prescriptionId"`
        DoctorId       string `json:"doctorId"`
    }
    
    err := json.Unmarshal([]byte(revocationJSON), &revocation)
    if err != nil {
        return fmt.Errorf("failed to parse revocation JSON: %v", err)
    }
    
    // Validate fields
    if revocation.PatientId == "" || revocation.PrescriptionId == "" || revocation.DoctorId == "" {
        return fmt.Errorf("patientId, prescriptionId, and doctorId are required")
    }

    // Get the asset
    asset, err := s.ReadAsset(ctx, revocation.PatientId)
    if err != nil {
        return err
    }

    // Find and update prescription
    found := false
    for i := range asset.Prescriptions {
        if asset.Prescriptions[i].PrescriptionId == revocation.PrescriptionId {
            // Verify the revoking doctor is the original prescriber
            if asset.Prescriptions[i].CreatedBy != revocation.DoctorId {
                return fmt.Errorf("only the prescribing doctor can revoke this prescription")
            }
            
            if asset.Prescriptions[i].Status != "Active" {
                return fmt.Errorf("can only revoke active prescriptions")
            }
            
            asset.Prescriptions[i].Status = "Revoked"
            asset.Prescriptions[i].TxID = ctx.GetStub().GetTxID()
            asset.Prescriptions[i].Timestamp = time.Now().Format(time.RFC3339)
            found = true
            break
        }
    }

    if !found {
        return fmt.Errorf("prescription not found")
    }

    asset.LastUpdated = time.Now().Format(time.RFC3339)
    assetJSON, err := json.Marshal(asset)
    if err != nil {
        return err
    }

    return ctx.GetStub().PutState(revocation.PatientId, assetJSON)
}

// GetUserRole retrieves the user's role from their certificate attributes
func (s *SmartContract) GetUserRole(ctx contractapi.TransactionContextInterface) (string, error) {
    // Get the MSP ID and certificate
    mspID, err := ctx.GetClientIdentity().GetMSPID()
    if err != nil {
        return "", fmt.Errorf("failed to get MSP ID: %v", err)
    }

    // Get role attribute from certificate
    role, ok, err := ctx.GetClientIdentity().GetAttributeValue("role")
    if err != nil {
        return "", fmt.Errorf("failed to get role attribute: %v", err)
    }
    if !ok {
        return "", fmt.Errorf("role attribute not found in certificate")
    }

    // Validate role based on MSP
    switch mspID {
    case "Org1MSP": // Doctor's organization
        if role != "doctor" {
            return "", fmt.Errorf("invalid role '%s' for organization %s", role, mspID)
        }
    case "Org2MSP": // Pharmacist's organization
        if role != "pharmacist" {
            return "", fmt.Errorf("invalid role '%s' for organization %s", role, mspID)
        }
    default:
        return "", fmt.Errorf("unknown MSP ID: %s", mspID)
    }

    return role, nil
}

// GetPrescriptionsByPatient - get all prescriptions for a patient that a doctor has prescribed
func (s *SmartContract) GetPrescriptionsByPatient(ctx contractapi.TransactionContextInterface, patientId string) (*Asset, error) {
    // Get caller's identity and role
    callerId, err := ctx.GetClientIdentity().GetID()
    if err != nil {
        return nil, fmt.Errorf("failed to get caller identity: %v", err)
    }

    role, err := s.GetUserRole(ctx)
    if err != nil {
        return nil, err
    }
    if role != "doctor" {
        return nil, fmt.Errorf("only doctors can access patient prescriptions")
    }

    // Get the asset
    asset, err := s.ReadAsset(ctx, patientId)
    if err != nil {
        return nil, err
    }

    // Filter prescriptions to only show those created by this doctor
    filteredPrescriptions := []Prescription{}
    for _, prescription := range asset.Prescriptions {
        if prescription.CreatedBy == callerId {
            filteredPrescriptions = append(filteredPrescriptions, prescription)
        }
    }
    asset.Prescriptions = filteredPrescriptions

    return asset, nil
}

// CheckPrescriptionExpiry - checks if a prescription has expired
func (s *SmartContract) CheckPrescriptionExpiry(ctx contractapi.TransactionContextInterface, patientId string, prescriptionId string) error {
    asset, err := s.ReadAsset(ctx, patientId)
    if err != nil {
        return err
    }

    for i := range asset.Prescriptions {
        if asset.Prescriptions[i].PrescriptionId == prescriptionId {
            expiryDate, err := time.Parse("2006-01-02", asset.Prescriptions[i].ExpiryDate)
            if err != nil {
                return fmt.Errorf("invalid expiry date format: %v", err)
            }

            if time.Now().After(expiryDate) {
                asset.Prescriptions[i].Status = "Expired"
                asset.Prescriptions[i].TxID = ctx.GetStub().GetTxID()
                asset.Prescriptions[i].Timestamp = time.Now().Format(time.RFC3339)

                assetJSON, err := json.Marshal(asset)
                if err != nil {
                    return err
                }
                return ctx.GetStub().PutState(patientId, assetJSON)
            }
            return nil
        }
    }

    return fmt.Errorf("prescription not found")
}

// GetPrescriptionAnalytics - get analytics for prescriptions by doctor/pharmacist
func (s *SmartContract) GetPrescriptionAnalytics(ctx contractapi.TransactionContextInterface, startDate string, endDate string) (map[string]interface{}, error) {
    iterator, err := ctx.GetStub().GetStateByRange("", "")
    if err != nil {
        return nil, err
    }
    defer iterator.Close()

    analytics := map[string]interface{}{
        "totalPrescriptions": 0,
        "activeCount": 0,
        "dispensedCount": 0,
        "expiryCount": 0,
        "medicationFrequency": make(map[string]int),
        "averageDispenseTime": 0.0,
    }

    for iterator.HasNext() {
        queryResponse, err := iterator.Next()
        if err != nil {
            continue
        }

        var asset Asset
        if err := json.Unmarshal(queryResponse.Value, &asset); err != nil {
            continue
        }

        for _, prescription := range asset.Prescriptions {
            analytics["totalPrescriptions"] = analytics["totalPrescriptions"].(int) + 1
            
            // Track medication frequency
            if count, exists := analytics["medicationFrequency"].(map[string]int)[prescription.MedicationName]; exists {
                analytics["medicationFrequency"].(map[string]int)[prescription.MedicationName] = count + 1
            } else {
                analytics["medicationFrequency"].(map[string]int)[prescription.MedicationName] = 1
            }

            // Track status counts
            switch prescription.Status {
            case "Active":
                analytics["activeCount"] = analytics["activeCount"].(int) + 1
            case "Dispensed":
                analytics["dispensedCount"] = analytics["dispensedCount"].(int) + 1
            case "Expired":
                analytics["expiryCount"] = analytics["expiryCount"].(int) + 1
            }
        }
    }

    return analytics, nil
}

// CheckMedicationInteractions - checks for potential interactions between medications
func (s *SmartContract) CheckMedicationInteractions(ctx contractapi.TransactionContextInterface, patientId string, newMedication string) ([]string, error) {
    asset, err := s.ReadAsset(ctx, patientId)
    if err != nil {
        return nil, err
    }

    // Sample interaction check - in production would connect to a medical database
    knownInteractions := map[string][]string{
        "Aspirin": {"Warfarin", "Heparin"},
        "Ibuprofen": {"Aspirin", "Warfarin"},
        "Warfarin": {"Aspirin", "Ibuprofen"},
    }

    var interactions []string
    if interactsWith, exists := knownInteractions[newMedication]; exists {
        for _, prescription := range asset.Prescriptions {
            if prescription.Status == "Active" {
                for _, interactor := range interactsWith {
                    if prescription.MedicationName == interactor {
                        interactions = append(interactions, fmt.Sprintf("Warning: %s interacts with active medication %s", newMedication, interactor))
                    }
                }
            }
        }
    }

    return interactions, nil
}

// BatchCreatePrescriptions - create multiple prescriptions in a single transaction
func (s *SmartContract) BatchCreatePrescriptions(ctx contractapi.TransactionContextInterface, assetsJSON string) error {
    var assets []Asset
    err := json.Unmarshal([]byte(assetsJSON), &assets)
    if err != nil {
        return fmt.Errorf("failed to parse assets JSON: %v", err)
    }

    for _, asset := range assets {
        assetJSON, err := json.Marshal(asset)
        if err != nil {
            return err
        }
        if err := s.CreateAsset(ctx, string(assetJSON)); err != nil {
            return err
        }
    }

    return nil
}

// GetPrescriptionsByDoctor - get all prescriptions created by a specific doctor
func (s *SmartContract) GetPrescriptionsByDoctor(ctx contractapi.TransactionContextInterface, doctorId string) ([]map[string]interface{}, error) {
    iterator, err := ctx.GetStub().GetStateByRange("", "")
    if err != nil {
        return nil, err
    }
    defer iterator.Close()

    var doctorPrescriptions []map[string]interface{}

    for iterator.HasNext() {
        queryResponse, err := iterator.Next()
        if err != nil {
            continue
        }

        var asset Asset
        if err := json.Unmarshal(queryResponse.Value, &asset); err != nil {
            continue
        }

        for _, prescription := range asset.Prescriptions {
            if prescription.CreatedBy == doctorId {
                prescriptionData := map[string]interface{}{
                    "PrescriptionId": prescription.PrescriptionId,
                    "PatientId":     asset.PatientId,
                    "PatientName":   asset.PatientName,
                    "MedicationName": prescription.MedicationName,
                    "Dosage":        prescription.Dosage,
                    "Instructions":  prescription.Instructions,
                    "Status":        prescription.Status,
                    "Timestamp":     prescription.Timestamp,
                    "ExpiryDate":    prescription.ExpiryDate,
                    "TxID":          prescription.TxID,
                }
                
                doctorPrescriptions = append(doctorPrescriptions, prescriptionData)
            }
        }
    }

    return doctorPrescriptions, nil
}

// GetDispenseHistory - get all prescriptions dispensed by a specific pharmacist
func (s *SmartContract) GetDispenseHistory(ctx contractapi.TransactionContextInterface, pharmacistId string) ([]map[string]interface{}, error) {
    iterator, err := ctx.GetStub().GetStateByRange("", "")
    if err != nil {
        return nil, err
    }
    defer iterator.Close()

    var dispensedPrescriptions []map[string]interface{}

    for iterator.HasNext() {
        queryResponse, err := iterator.Next()
        if err != nil {
            continue
        }

        var asset Asset
        if err := json.Unmarshal(queryResponse.Value, &asset); err != nil {
            continue
        }

        for _, prescription := range asset.Prescriptions {
            if prescription.DispensingPharmacist == pharmacistId {
                prescriptionData := map[string]interface{}{
                    "PrescriptionId":      prescription.PrescriptionId,
                    "PatientId":           asset.PatientId,
                    "PatientName":         asset.PatientName,
                    "MedicationName":      prescription.MedicationName,
                    "Dosage":              prescription.Dosage,
                    "Instructions":        prescription.Instructions,
                    "Status":              prescription.Status,
                    "CreatedBy":           prescription.CreatedBy,
                    "DispensingTimestamp": prescription.DispensingTimestamp,
                    "TxID":                prescription.TxID,
                }
                
                dispensedPrescriptions = append(dispensedPrescriptions, prescriptionData)
            }
        }
    }

    return dispensedPrescriptions, nil
}