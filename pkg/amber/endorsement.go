// Copyright 2022 The Project Oak Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package amber

import (
	"encoding/json"
	"fmt"
	"os"

	intoto "github.com/in-toto/in-toto-golang/in_toto"
)

// AmberEndorsementV2 is the ClaimType for Amber Endorsements V2. This is
// expected to be used together with the AmberClaimV1 as the predicate type in
// an in-toto statement. This version of Amber Endorsement replaces the earlier
// version in `schema/amber-endorsement/v1`.
const AmberEndorsementV2 = "https://github.com/project-oak/transparent-release/endorsement/v2"

// ParseEndorsementV2File reads a JSON file from the given path, and parses it into an
// instance of intoto.Statement, with the Amber Claim as the build type.
func ParseEndorsementV2File(path string) (*intoto.Statement, error) {
	statementBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read the endorsement file: %v", err)
	}

	var statement intoto.Statement
	if err := json.Unmarshal(statementBytes, &statement); err != nil {
		return nil, fmt.Errorf("could not unmarshal the endorsement file:\n%v", err)
	}

	// statement.Predicate is now just a map, we have to parse it into an instance of ClaimPredicate.
	predicateBytes, err := json.Marshal(statement.Predicate)
	if err != nil {
		return nil, fmt.Errorf("could not marshal Predicate map into JSON bytes: %v", err)
	}

	var predicate ClaimPredicate
	if err = json.Unmarshal(predicateBytes, &predicate); err != nil {
		return nil, fmt.Errorf("could not unmarshal JSON bytes into a slsa.ProvenancePredicate: %v", err)
	}

	// Replace the Predicate map with ClaimPredicate
	statement.Predicate = predicate

	if err = validateAmberClaim(statement); err != nil {
		return nil, fmt.Errorf("the predicate in the endorsement file is invalid: %v", err)
	}

	return &statement, nil
}

func validateAmberClaim(statement intoto.Statement) error {
	predicate, err := ValidateAmberClaim(statement)
	if err != nil {
		return err
	}

	if predicate.ClaimType != AmberEndorsementV2 {
		return fmt.Errorf(
			"the predicate does not have the expected claim type; got: %s, want: %s",
			predicate.ClaimType,
			AmberEndorsementV2)
	}

	return nil
}
