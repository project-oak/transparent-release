# Claim Transparency

The following diagram shows the journey that software takes from code to a deployable application
used by an end user (either as an application deployed locally or as a remote server). During this
process several artifacts (e.g., code, software libraries, and binaries) are generated and
transformed into another (e.g., through compiling and linking). The premise of software supply
chain security is that many things could go wrong during this process, either due to human error
or attacks on the software supply chain by malicious actors.

![The journey of a software binary](images/journey.png)

To provide assurances to the end users about the security and privacy of a software application, in
the transparent release project our goal is to provide transparency into the build and release
processes. In our solution, in each step of the process software artifacts are being reviewed and
audited and the result, a claim about the security and privacy properties of the software artifact,
is signed and published into a [transparency log](https://continusec.com/static/VerifiableDataStructures.pdf).
The goal is to make these claims easily discoverable so that anyone can use the claims in the
assessment of privacy policies.

## The Claim Format

We define the following structure, based on the [in-toto Statement standard](https://github.com/in-toto/attestation/blob/main/spec/README.md#statement),
for specifying security and privacy claims. This format is meant to be generic and allow specifying
different types of claims.

```json
{
  "_type": "https://in-toto.io/Statement/v0.1",
  "subject": [{ ... }],

  "predicateType": "https://github.com/project-oak/transparent-release/schema/claim/v1",
  "predicate": {
    "claimType": "<URI>",
    "issuedOn": "<TIMESTAMP>",
    "validity": {
      "notBefore": "<TIMESTAMP>",
      "notAfter": "<TIMESTAMP>",
    },
    "claimSpec": { /* object */ },
    "evidence": [
      {
        "role": "<STRING>",
        "uri": "<URI>",
        "digest": { /* DigestSet */ }
      }
    ]
  }
}
```
Section [Examples](##Examples) demonstrates the customization and use of the claim format via a
number of examples.

### Fields

This section describes the semantics of each field in the claim format:

- **subject** _(array of objects, required)_:
  Set of artifacts (e.g., source code, or some binary) that the claim applies to.
  - **subject[*].digest** and **subject[*].name** as defined by Statement in the in-toto standard.
- **claimType** _(string ([TypeURI](https://github.com/in-toto/attestation/blob/main/spec/field_types.md#TypeURI)), required)_:
  URI indicating what type of claim was issued. It determines the meaning of claimSpec and evidence below.
- **issuedOn** _(string ([Timestamp](https://github.com/in-toto/attestation/blob/main/spec/field_types.md#Timestamp)), required)_:
    The timestamp at which this claims was generated.
- **validity** _(object, required)_:
  Validity duration of the claim.

  - **validity.notBefore** _(string ([Timestamp](https://github.com/in-toto/attestation/blob/main/spec/field_types.md#Timestamp)), required)_:
    The timestamp from which the claim is effective, and the artifact is endorsed for use. Must be
    equal or after the issuedOn timestamp.
  - **validity.notAfter** _(string ([Timestamp](https://github.com/in-toto/attestation/blob/main/spec/field_types.md#Timestamp)), required)_:
    The timestamp of when the artifact is no longer endorsed for use. This, combined with the
    `notBefore` field, is particularly useful for implementing passive revocation.

- **claimSpec** _(object, optional)_:
  Gives a detailed description of the claim, and the steps that were taken to perform the assessment
  of the artifact in the subject. This is an arbitrary JSON object with a schema defined by
  claimType. Depending on the claimType, the claimSpec could be anything, including:

  - A free-text description of the claim and the review/audit process. A certain type of claim with
    a more detailed schema for claimSpec may explicitly capture such details as the scope,
    limitations & threats, and the additional material that was used to perform the assessment, for
    instance design docs may be used as one input. Such materials should not be included in the
    subject, unless the review is particularly about that material (e.g., a design doc).
  - The content of a [cargo-crev](https://github.com/crev-dev/cargo-crev) review, or an
    [audit recorded via cargo vet](https://mozilla.github.io/cargo-vet/recording-audits.html).
  - A link to a report similar to this security review of a cryptographic Rust crate, identified by
    its URI and digest.
  - An auto-generated report, for instance a fuzz testing report from ClusterFuzz.
  - A [datasheet about a dataset](https://arxiv.org/abs/1803.09010).

- **evidence** _(array of objects, optional)_:
  The collection of artifacts that were generated during the assessment to support the claim, or
  existing claims that were assumed to be true, and were used as input to the assessment process.
  Some examples of evidence include:

  - Provenance
  - Reports from executed test suites
  - List of other claims about the dependencies, build materials, etc.
  - Audits of earlier versions of the same artifact (e.g., source code). For instance if an earlier
    version had a rigorous external audit, for a new revision, the audit/review could focus on the
    diff (e.g., cargo has a feature for it: `review --diff`). A suite of regression tests or
    security analysis tools dedicated to checking specific security properties, could be very
    useful in such cases.

  The reliance on the evidence is not quantified. So there is not a field for stating the level of
  trustworthiness or relevance for a piece of evidence. Instead, all included pieces of evidence
  are treated the same. Note that claimSpec may still distinguish between them based on their roles.

  - **evidence[*].role** _(string, required)_:
    This field is used to specify the type and role of the evidence within the claim. The meaning
    of it is specified by claimType and within the context of claimSpec.
  - **evidence[*].uri** _(string ([ResourceURI](https://github.com/in-toto/attestation/blob/main/spec/field_types.md#ResourceURI)), required)_:
    An evidence could be another claim (possibly of another claimType) or a report publicly
    available from a URI. Either way, the URI should be provided in this field.
  - **evidence[*].digest** _(object ([DigestSet](https://github.com/in-toto/attestation/blob/main/spec/field_types.md#DigestSet)), required)_:
    Collection of cryptographic digests for the contents of this artifact.

## Comparison to the SLSA provenance format

The following table shows the correspondence between the fields in a claim statement as described
above, and a [SLSA provenance statement](https://slsa.dev/provenance/v0.2). Note that the table
does not provide a correspondence between all fields. Rather, the goal is to show that the two
formats follow the same design principles. In particular, to support flexibility, via
buildType/buildConfig, and claimType/claimSpec; and to allow linking of related materials/evidence.
The table does not intend to suggest that one format could replace the other, as the two formats
are conceptually different. For instance, the SLSA provenance format has an invocations field,
which is meaningless if the format were to be used for specifying a security or privacy claim.
Builder and buildConfig are other fields that are irrelevant to security or privacy claims.
Similarly the field names in the schema suggested for claims are meaningless in the context of a
provenance statement.

| Field in a Claim statement | Field in a SLSA provenance | Comments |
|:----------------|:---------------|:-----------------------------------------------------------|
| claimType | buildType | Both define the meanings of the other fields in the predicate.|
| claimSpec | buildConfig | Both provide a flexible way of supporting different types of content (claims, and build processes).|
| evidence | materials | Optional list of (a subset of ) additional artifacts that influenced the statement. |

## Comparison to RATS

The Remote ATtestation procedureS (RATS) working group has provided an [architecture](https://datatracker.ietf.org/doc/html/draft-ietf-rats-architecture)
and glossary of concepts related to remote attestation. [This cheatsheet](https://github.com/thomas-fossati/rats-cheatsheet)
and [this slides deck](https://confidentialcomputing.io/wp-content/uploads/sites/85/2021/09/IETF-Remote-Attestation-Architecture-Overview.pdf)
give an overview of the architecture and the main concepts. RATS has many concepts similar
to the ones in our design, but seems to be focused on claims and evidence that are generated and
consumed automatically. Claims and evidence in RATS are designed to be used for remote attestation.
The claims in our binary transparency ecosystem, however, are not limited to the ones used for
remote attestation. We target a wider range of use cases.

## Examples

### Endorsement Claims

With the schema given for claims, an endorsement statement can be seen as a claim where the
claimSpec is empty. In this case, the claimType has to clearly state that the claim is an
endorsement statement. An endorsement statement, in the evidence field, may provide links to a
provenance statement. In this case, the provenance statement must have the same subject as the one
in the endorsement statement. If the linked provenance is signed, the evidence list may as well
include a reference to a Rekor log entry corresponding to the provenance.

```json
{
  "_type": "https://in-toto.io/Statement/v0.1",
  "subject": [
    {
      "name": "oak_functions-012a5206e5ab35d2778832638519441dd27664da",
      "digest": {
        "sha256": "01b792106ef1f61eece3a666ac6069875fc90b942fefc3fe931f016395bb6c88"
      }
    }
  ],

  "predicateType": "https://github.com/project-oak/transparent-release/schema/claim/v1",
  "predicate": {
    "claimType": "https://gh/project-oak/transparent-release/schema/endorsement/v2",
    "issuedOn": "2022-06-08T10:20:50.32Z",
    "validity": {
      "notBefore": "2022-06-08T10:20:50.32Z",
      "notAfter": "2022-06-09T10:20:50.32Z"
    },
    "evidence": [
      {
        "role": "Provenance",
        "uri": "https://gh/project-oak/oak/blob/provenance/<bin-hash>/<commit-hash>.json",
        "digest": {
          "sha256": "<provenance file sha256 hash>"
        }
      }
    ]
  }
}
```

A more sophisticated claimType for endorsements would have a non-empty claimSpec, containing a
specification of the policy that was checked before issuing the endorsement statement.
Authorization logic is a good candidate for providing a specification of such a policy. In this
case the tool that verified the policy and generated the claim will as well sign the claim. 

```json
{
  "_type": "https://in-toto.io/Statement/v0.1",
  "subject": [{
    "name": "oak_functions-012a5206e5ab35d2778832638519441dd27664da",
    "digest": {
      "sha256": "01b792106ef1f61eece3a666ac6069875fc90b942fefc3fe931f016395bb6c88"
    }
  }],

  "predicateType": "https://gh/project-oak/transparent-release/schema/claim/v1",
  "predicate": {
    "claimType": "https://gh/project-oak/transparent-release/schema/endorsement/v3",
    "issuedOn": "2022-06-08T10:20:50.32Z",
    "validity": {
      "notBefore": "2022-06-08T10:20:50.32Z",
      "notAfter": "2022-06-09T10:20:50.32Z",
    },
    "claimSpec": {
      "verification": "<The provenance verification policy, in authorization logic, that was verified as a precondition for issuing this endorsement statement.>",
      ...
    },
    "evidence": [
      {
        "role": "Provenance",
        "uri": "https://gh/project-oak/oak/blob/provenance/<bin-hash>/<commit-hash>.json",
        "digest": {
          "sha256": "<provenance file sha256 hash>"
        }
      }
    ]

  }
}

```

### Free-text Claims

For most use-cases, we can allow a claimType with a claimSpec containing free-format text. The text
could be a short sentence like the following about the Oak Functions trusted runtime, or a more
elaborate description, or a link to a full report, ideally identified by the hash of the content of
the report.

```json
{
  "_type": "https://in-toto.io/Statement/v0.1",
  "subject": [
    {
      "name": "oak_functions",
      "digest": {
        "sha1": "012a5206e5ab35d2778832638519441dd27664da"
      }
    }
  ],

  "predicateType": "https://gh/project-oak/transparent-release/schema/claim/v1",
  "predicate": {
    "claimType": "https://github.com/project-oak/oak/claim/v1",
    "issuedOn": "2022-06-08T10:20:50.32Z",
    "validity": {
      "notBefore": "2022-06-08T10:20:50.32Z",
      "notAfter": "2022-06-09T10:20:50.32Z"
    },
    "claimSpec": "Oak trusted runtime does not store or log any parts of the incoming request",
    "evidence": [
      {
        "role": "Source-Code",
        "uri": "https://github.com/project-oak/oak",
        "digest": {
          "sha1": "012a5206e5ab35d2778832638519441dd27664da"
        }
      }
    ]
  }
}
```

### Auto-generated Claims

One type of auto-generated claim about a source code could be a fuzz testing report generated by an
[OSS-Fuzz](https://github.com/google/oss-fuzz) project. Such a claim may include coverage report as
shown in the following example.

```json
{
  "_type": "https://in-toto.io/Statement/v0.1",
  "subject": [{
    "name": "oak_functions",
    "digest": {
      "sha1": "012a5206e5ab35d2778832638519441dd27664da"
  },

  "predicateType": "https://gh/project-oak/transparent-release/schema/claim/v1",
  "predicate": {
    "claimType": "https://gh/project-oak/transparent-release/schema/fuzz_report/v1",
    "issuedOn": "2022-06-08T10:20:50.32Z",
    "validity": {
      "notBefore": "2022-06-08T10:20:50.32Z",
      "notAfter": "2022-06-09T10:20:50.32Z"
    },
    "claimSpec": {
      "fuzz_target": {
        "name": "libFuzzer_oak_apply_policy",
        "uri": "https://gh/project-oak/oak/blob/56c9603bee7feb7928fd1d8a16d547badfe7ae8f/oak_functions/loader/fuzz/fuzz_targets/apply_policy.rs"
      },
      "coverage": {
        "line": 3.68% (5223/142079),
        "function": 3.61% (729/20172),
        "region": 2.55% (1795/70407)
      },
      "failed": false,
    },
    "evidence": [
      {
        "role": "cluster-fuzz report",
        "uri": "https://oss-fuzz.com/performance-report/libFuzzer_oak_apply_policy/libfuzzer_asan_oak/2022-06-07",
        "digest": {
          "sha256": "<sha256 hash of a zip file containing the logs>"
        }
      }
    ]
  }
}
```
