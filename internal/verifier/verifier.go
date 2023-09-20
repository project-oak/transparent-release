// Copyright 2022-2023 The Project Oak Authors
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

package verifier

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/project-oak/transparent-release/internal/model"
	pb "github.com/project-oak/transparent-release/pkg/proto/verifier"
	"go.uber.org/multierr"
	"google.golang.org/protobuf/encoding/prototext"
)

// Verify checks that the provenance conforms to expectations, returning a
// list of errors whenever the verification failed.
//
//nolint:cyclop,gocognit,gocyclo
func Verify(provenances []model.ProvenanceIR, verOpts *pb.VerificationOptions) error {
	if provenances == nil {
		panic(fmt.Errorf("provenances must not be nil"))
	}

	var errs error

	if verOpts.ProvenanceCountAtLeast != nil && len(provenances) < int(verOpts.ProvenanceCountAtLeast.Count) {
		errs = multierr.Append(errs, fmt.Errorf("too few provenances: have %d but want at least %d", len(provenances), verOpts.ProvenanceCountAtLeast.Count))
	}

	if verOpts.ProvenanceCountAtMost != nil && len(provenances) > int(verOpts.ProvenanceCountAtMost.Count) {
		errs = multierr.Append(errs, fmt.Errorf("too many provenances: have %d but want at most %d", len(provenances), verOpts.ProvenanceCountAtMost.Count))
	}

	if verOpts.AllSameBinaryName != nil && len(provenances) > 1 {
		expectedBinaryName := provenances[0].BinaryName()
		for _, p := range provenances {
			if p.BinaryName() != expectedBinaryName {
				errs = multierr.Append(errs, fmt.Errorf("not all have same binary name"))
			}
		}
	}

	if verOpts.AllSameBinaryDigest != nil && len(provenances) > 1 {
		expectedDigest := provenances[0].BinarySHA256Digest()
		for _, p := range provenances {
			if p.BinarySHA256Digest() != expectedDigest {
				errs = multierr.Append(errs, fmt.Errorf("not all have same SHA2-256 binary digest"))
			}
		}
	}

	if verOpts.AllWithBuildCommand != nil {
		for i, p := range provenances {
			if buildCmd, err := p.BuildCmd(); err != nil || len(buildCmd) == 0 {
				errs = multierr.Append(errs, fmt.Errorf("no build command found in #%d", i))
			}
		}
	}

	if verOpts.AllWithBinaryName != nil {
		for i, p := range provenances {
			if p.BinaryName() != verOpts.AllWithBinaryName.BinaryName {
				errs = multierr.Append(errs, fmt.Errorf("unexpected binary name in #%d: got %q but want %q", i, p.BinaryName(), verOpts.AllWithBinaryName.BinaryName))
			}
		}
	}

	if verOpts.AllWithBinaryDigests != nil {
		for index, provenance := range provenances {
			digest := provenance.BinarySHA256Digest()
			found := false
			for _, digests := range verOpts.AllWithBinaryDigests.Digests {
				for f, d := range digests.Binary {
					if f != int32(pb.Digest_SHA2_256) {
						continue
					}
					if digest == hex.EncodeToString(d) {
						found = true
						break
					}
				}
				if found {
					break
				}
				for f, d := range digests.Hexadecimal {
					if f != int32(pb.Digest_SHA2_256) {
						continue
					}
					if digest == d {
						found = true
						break
					}
				}
			}
			if !found {
				errs = multierr.Append(errs, fmt.Errorf("could not match binary digest in #%d: %q", index, digest))
			}
		}
	}

	if verOpts.AllWithRepository != nil {
		expected := verOpts.AllWithRepository.RepositoryUri
		for index, provenance := range provenances {
			repoURI := ""
			if provenance.HasRepoURI() {
				repoURI = provenance.RepoURI()
			}
			if repoURI != expected {
				errs = multierr.Append(errs, fmt.Errorf("repository mismatch in #%d: got %q but want %q", index, repoURI, expected))
			}
		}
	}

	if verOpts.AllWithBuilderNames != nil {
		for index, provenance := range provenances {
			buiilderName, err := provenance.TrustedBuilder()
			if err != nil {
				buiilderName = ""
			}
			found := false
			for _, name := range verOpts.AllWithBuilderNames.BuilderNames {
				if buiilderName == name {
					found = true
					break
				}
			}
			if !found {
				errs = multierr.Append(errs, fmt.Errorf("could not match builder name in #%d: %q", index, buiilderName))
			}
		}
	}

	if verOpts.AllWithBuilderDigests != nil {
		for index, provenance := range provenances {
			digest, err := provenance.BuilderImageSHA256Digest()
			if err != nil {
				digest = ""
			}
			found := false
			for _, digests := range verOpts.AllWithBuilderDigests.Digests {
				for f, d := range digests.Binary {
					if f != int32(pb.Digest_SHA2_256) {
						continue
					}
					if digest == hex.EncodeToString(d) {
						found = true
						break
					}
				}
				if found {
					break
				}
				for f, d := range digests.Hexadecimal {
					if f != int32(pb.Digest_SHA2_256) {
						continue
					}
					if digest == d {
						found = true
						break
					}
				}
			}
			if !found {
				errs = multierr.Append(errs, fmt.Errorf("could not match builder digest in #%d: %q", index, digest))
			}
		}
	}

	return errs
}

// LoadVerificationOptions loads VerificationOptions from a textproto file.
func LoadVerificationOptions(path string) (*pb.VerificationOptions, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file from %q: %v", path, err)
	}
	return ParseVerificationOptions(string(bytes))
}

// LoadVerificationOptions parses VerificationOptions from textproto.
func ParseVerificationOptions(textproto string) (*pb.VerificationOptions, error) {
	var opts pb.VerificationOptions
	if err := prototext.Unmarshal([]byte(textproto), &opts); err != nil {
		return nil, fmt.Errorf("parse VerificationOptions: %v", err)
	}
	return &opts, nil
}
