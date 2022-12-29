# FuzzBinder Design

## Objective

FuzzBinder generates a _FuzzingClaim_ for a revision of a source code, based on Fuzz reports from
OSS-Fuzz.

## Background

FuzzingClaims can be included as a new type of evidence in
[endorsement claims](claim-transparency.md#endorsement-claims). FuzzingClaims can help reason about
the security of a binary.

Fuzzing is an automated testing technique for vulnerability detection using generated malformed
inputs to trigger unwanted behaviors to find bugs in binaries. Fuzzing statistics and metrics can be
automatically generated, and are good candidates to reason about the security of a binary given that
enough fuzzing effort is spent.

[OSS-Fuzz](https://github.com/google/oss-fuzz) is a continuous fuzzing platform for open source
software. It is used to fuzz about 650 open source projects. This platform is useful in this context
since it generates ~2MB of metadata per project per day per fuzzer to give insights to the
developers on how to improve the fuzz-targets of their open source projects to detect more bugs.
Indeed, this automatically generated metadata is useful for extracting statistics and fuzzing
metrics that can be included in fuzzing claims.

## Design

### FuzzClaim format

In this section, a customization of the [Claim Format](claim-transparency.md#the-claim-format) for
specifying fuzzing claims is presented.

```json
{
  "_type": "https://in-toto.io/Statement/v0.1",
  "subject": [
    {
      "name": "https://github.com/project-oak/oak",
      "digest": {
        "sha1": "012a5206e5ab35d2778832638519441dd27664da"
      }
    }
  ],

  "predicateType": "https://github.com/project-oak/transparent-release/claim/v1",
  "predicate": {
    "claimType": "https://github.com/project-oak/transparent-release/fuzz_claim/v1",
    "issuedOn": "2022-06-08T10:20:50.32Z",
    "validity": {
      "notBefore": "2022-06-08T10:20:50.32Z",
      "notAfter": "2022-06-09T10:20:50.32Z"
    },
    "claimSpec": {
      "perTarget": [
        {
          "name": "<some-fuzz-target>",
          "path": "fuzz/fuzz_targets/<some-fuzz-target>.rs",
          "fuzzStats": {
            "lineCoverage": "3.68% (5223/142079)",
            "branchCoverage": "3.61% (729/20172)",
            "detectedCrashes": false,
            "fuzzTimeSeconds": 9865.54,
            "numberFuzzTests": 463398
          }
        },
        {
          "name": "<some-other-fuzz-target>",
          "path": "fuzz/fuzz_targets/<some-other-fuzz-target>.rs",
          "fuzzStats": {
            "lineCoverage": "2.63% (3743/142079)",
            "branchCoverage": "23.06% (4653/20172)",
            "detectedCrashes": false,
            "fuzzTimeSeconds": 4525.45,
            "numberFuzzTests": 856398
          }
        }
      ],
      "perProject": {
        "lineCoverage": "4.59% (6523/142079)",
        "branchCoverage": "3.61% (729/20172)",
        "detectedCrashes": false,
        "fuzzTimeSeconds": 14390.99,
        "numberFuzzTests": 1319796,
        "sanitizers": ["asan"],
        "fuzzEngines": ["libFuzzer"]
      }
    },
    "evidence": [
      {
        "role": "project coverage",
        "uri": "<ent file uri>",
        "digest": {
          "sha256": "<sha256 of the project coverage summary.json>"
        }
      },
      {
        "role": "fuzzTarget coverage",
        "uri": "<ent file uri>",
        "digest": {
          "sha256": "<sha256 of <some-fuzz-target> coverage summary.json>"
        }
      },
      ,
      {
        "role": "fuzzTarget coverage",
        "uri": "<ent file uri>",
        "digest": {
          "sha256": "<sha256 of <some-other-fuzz-target> coverage summary.json>"
        }
      },
      {
        "role": "srcmap",
        "uri": "<ent file uri>",
        "digest": {
          "sha256": "<sha256 of the srcmap file linking the date to the revision of the source code>"
        }
      }
    ]
  }
}
```

The semantics of each field of the general Claim Format are described in the
[Claim Format documentation](claim-transparency.md#fields). In the next paragraph, a description of
the customized fields (defined in FuzzClaim format) is provided.

- **\_type** (string (_TypeURI_), required): as defined by Statement in the
  [in-toto standard](https://github.com/in-toto/attestation/blob/main/spec/README.md#statement).
- **subject** (array of objects, required): as defined in the
  [Claim Format](claim-transparency.md#fields).
- **predicateType** (string (_TypeURI_), required): as defined by Statement in the
  [in-toto standard](https://github.com/in-toto/attestation/blob/main/spec/README.md#statement).
- **claimType** (string (_TypeURI_), required): as defined in the
  [Claim Format](claim-transparency.md#fields).
- **issuedOn** (string (_Timestamp_), required): as defined in the
  [Claim Format](claim-transparency.md#fields).
- **validity** (object, required): as defined in the [Claim Format](claim-transparency.md#fields).
- **claimSpec** (object, required): gives a detailed description of the fuzzing claims, and the
  needed metrics and statistics to characterize the security of the fuzzed revision of the source
  code.
  - **claimSpec.perTarget** (array of objects, required): an array of the fuzzing metrics and
    statistics for each fuzz-target.
    - **perTarget[*].name** (string, required): name of the fuzz-target.
    - **perTarget[*].path** (string, required): path of the fuzz-target, relative to the root of the
      git repository of the project.
    - **perTarget[*].fuzzStats.lineCoverage** (string, required): specifies line coverage by the
      fuzz-target.
    - **perTarget[*].fuzzStats.branchCoverage** (string, required): specifies branch coverage by the
      fuzz-target.
    - **perTarget[*].fuzzStats.detectedCrashes** (number, required): specifies if any crashes were
      detected by the fuzz-target.
    - **fuzzEffort[*].fuzzStats.fuzzTimeSeconds** (number, optional): specifies the fuzzing time in
      seconds.
    - **fuzzEffort[*].fuzzStats.numberFuzzTests** (number, optional): specifies the number of
      executed fuzzing tests.
  - **claimSpec.perProject** (object, required): an object of the fuzzing metrics and statistics for
    all the fuzz-targets aggregated.
    - **perProject.lineCoverage** (string, required): specifies line coverage by all fuzz-targets.
    - **perProject.branchCoverage** (string, required): specifies branch coverage by all
      fuzz-targets.
    - **perProject.detectedCrashes** (number, required): specifies the number of detected crashes
      using all fuzz-targets.
    - **perProject.fuzzTimeSeconds** (number, optional): specifies the fuzzing time in seconds.
    - **perProject.numberFuzzTests** (number, optional): specifies the number of executed fuzzing
      tests.
    - **fuzzEffort.sanitizers** (array of strings, required): specifies the list of the used
      sanitizers (as defined in the project configuration in OSS-Fuzz repository).
    - **fuzzEffort.fuzzEngines** (array of strings, required): specifies the list of used fuzzing
      engines (as defined in the project configuration in OSS-Fuzz repository).
- **evidence** (array of objects, required): as defined by the
  [Claim Format](claim-transparency.md#fields). It is the collection of the fuzzing reports that are
  used to generate the FuzzClaim.

### FuzzClaim specification

In this section, the metrics that are included in the `claimSpec` section of the FuzzClaim are
explained along with the reasons for their inclusion.

#### Bugs and crashes

Note that bugs and crashes are two different things in OSS-Fuzz terminology. Bugs are unique and
deterministic while crashes can be deduplicated.

The metric that is used in the fuzzing claims to characterize crashes is **whether a crash is
detected**.

It is also interesting to add the number of bugs in the future. However, the detection of bugs needs
more computational effort because it requires the analysis of several crashes.

#### Coverage

Note that the design ideas in this section make use of the recommendations (based on experiments) of
[this paper](https://www.sciencedirect.com/science/article/pii/S0167404822003388).

Even though the goal of fuzzing is to find new bugs or crashes, the number of bugs and crashes is
not a fine-grained metric to accurately judge the security of the revision of the source code,
especially when there are no bugs (i.e., the absence of bugs is a necessary but not sufficient
condition to endorse the security of a revision of the source code).

Coverage metrics can help assess the security of the revision of the source code because they are
rather fine-grained and can be easily inspected. These measures are intended to approximate the
amount of functionality in a program that is actually tested during fuzzing (i.e., they partially
assess the exploration process performed by fuzzers to detect bugs).

There are many coverage metrics available. For instance,
[OSS-Fuzz coverage reports](https://google.github.io/oss-fuzz/advanced-topics/code-coverage/) are
based on
[Clang code coverage](https://clang.llvm.org/docs/SourceBasedCodeCoverage.html#source-based-code-coverage)
which tracks
[five coverage statistics](https://clang.llvm.org/docs/SourceBasedCodeCoverage.html#interpreting-reports):
function coverage, instantiation coverage, line coverage, region coverage, and branch coverage. All
these statistics refer to a specific part of the target that has been executed at least once during
the fuzz testing.

In our use case, we target metrics that are fine-grained enough to provide reasonable indicators of
the proportion of code that has been tested. Ideally, they should be able to show how well
individual lines of the code have been tested beyond the simple yes/no answer. Achieving 100% line
coverage does not prove that all possible execution paths have been executed, and thus that all
potential bugs can be detected.

However, even though 100% line coverage does not mean 100% bug detection, very low line coverage
indicates a low ability to find bugs. Therefore, **line coverage** is used in FuzzBinder. But using
line coverage alone can be misleading since it does not take into account the algorithmic complexity
of the the source code and thus does not reflect how much of its logic is tested.

Therefore, **branch coverage** is used as a complementary metric to line coverage. Note that using
the branch coverage alone can be misleading as well since executing multiple branches with very few
lines of code gives the illusion that most of the code was executed. Therefore, it should be coupled
with line coverage.

The rest of the coverage metrics (function coverage, instantiation coverage, and region coverage)
are not used by FuzzBinder since they are less informative about the covered code and harder to
analyze or use to draw conclusions. For instance, function coverage does not indicate which part of
the function was covered.

#### Fuzzing effort

It is important to add the fuzzing efforts to the FuzzClaim specification since they have a direct
impact on the crash detection. This includes the **number of performed tests** and the **total
fuzzing time**.

### Usage of the FuzzClaim specification

The purpose of the FuzzClaim specification is not to judge the security of the revision of the
source code but to provide all the needed elements for users to characterize the security of the
revision of the source code.

It is up to the user of the FuzzClaim (a person, a team, or a trusted tool) to decide whether or not
to use the FuzzClaim as an evidence for security (e.g., in an endorsement statement). In particular,
the user has to decide how and whether to use FuzzClaims when certain crashes are detected, and to
define thresholds for line/branch coverage and fuzzing efforts. These decisions can be part of a
team's security policy or part of their release-readiness criteria.

### Scraper module (FuzzScraper)

FuzzScraper extracts fuzzing metrics from fuzzing reports generated by OSS-Fuzz. In the design of
FuzzScraper, we have to keep in mind that:

- OSS-Fuzz fuzzing reports are generated for every last commit of a given day (according to the
  current configuration). Therefore, not all the revisions of the source code are fuzzed.
- The coverage reports generated by OSS-Fuzz and ClusterFuzz, stored in their GCS buckets, are
  organized by date. Therefore, FuzzScraper needs to link the fuzzing date to the revision of the
  code that was used to generate them (the coverage reports) to extract the fuzzing data of that
  revision of the code. To link the date to the revision of the source code, two solutions can be
  considered. The first is to extract the revision hash from the coverage builds metadata and the
  second is to extract it from the fuzzers builds metadata. After considering both solutions, we
  chose the first one because the second one can lead to inconsistency, especially since different
  fuzzers can use several revisions of the source code on a given day
  [up to 4 builds](https://github.com/google/oss-fuzz/blob/master/docs/getting-started/new_project_guide.md#builds_per_day-optional-build_frequency)
  that can be different across fuzzers, while the coverage build uses exactly one revision of the
  source code (the same for all fuzzers).

  However, even if a revision of a source code is selected using the coverage build metadata, it is
  important to check that this selected revision is consistent across the fuzzers, otherwise the
  extracted coverage metrics may be misleading. In fact, if a revision of the source code is
  selected but is not actually used for one of the fuzzers on that day, a corpus linked to another
  revision of the source code will be used for the coverage report of that fuzzer (even though the
  corpus was not used by that fuzzer to fuzz the revision of the source code we selected). In the
  current FuzzClaim format, consistency can be checked by verifying that there is fuzzing efforts
  for all the fuzz-targets.

- The extracted information from fuzzing logs needs to be aggregated to get the total fuzzing effort
  per target or per project on a given day.

In the table below, the source from which each metric can be extracted is provided. These metrics
are used to generate the claimSpec of the FuzzClaims.

_Note that in some cases, the data extraction is not direct and some computation or data aggregation
is needed._

| Data to Extract                                                                | GCS Bucket                                                                                                                                                   | Use for FuzzBinder                                                                                  | Data Retention Period |
| :----------------------------------------------------------------------------- | :----------------------------------------------------------------------------------------------------------------------------------------------------------- | :-------------------------------------------------------------------------------------------------- | :-------------------- |
| Link the date to the revision of the code.                                     | `gs://oss-fuzz-coverage/{projectName}/srcmap/{date}.json`                                                                                                    | Link the date to the revision of the source code that was used for the coverage build on that date. | More than a year      |
| Extract the line and branch coverage (perProject).                             | `gs://oss-fuzz-coverage/{projectName}/reports/{date}/linux/summary.json`                                                                                     | Get the branch and line coverage for all the fuzz targets aggregated and link them to a given date. | More than a year      |
| Extract the line and branch coverage (perTarget).                              | `gs://oss-fuzz-coverage/{projectName}/fuzzer_stats/{date}/{fuzz-target}.json`                                                                                | Get the branch and line coverage per fuzz target and link them to a given date.                     | More than a year      |
| Extract and compute the fuzzing effort (perTarget and/or per Project) per day. | `gs://{projectName}-logs.clusterfuzz-external.appspot.com/{fuzzEngine}_{projectName}_{fuzz-target}/{fuzzengine}_{sanitizer}_{projectName}/{date}/{time}.log` | Compute the total fuzzing time in seconds and the number of executed tests on a given day.          | 15 days               |
| Detected crashes (perTarget and/or per Project) per day.                       | `gs://{projectName}-logs.clusterfuzz-external.appspot.com/{fuzzEngine}_{projectName}_{fuzz-target}/{fuzzengine}_{sanitizer}_{projectName}/{date}/{time}.log` | Detect the new crashes on a given day.                                                              | 15 days               |

### Claims evidence (FuzzEvidence)

The evidence is the collection of the fuzzing reports used to generate a FuzzClaim on a given date,
and help to reproduce the claim in the future if verification is needed. In FuzzBinder, the evidence
roles are:

- **“srcmap”**: refers to the role of the evidence files that help to map a given date to a revision
  of the source code (a commit hash). In FuzzBinder, we decided to use files in
  `gs://oss-fuzz-coverage/{projectName}/srcmap/{date}.json` for this purpose, since they guarantee
  that each date is linked to exactly one revision of the source code.
- **“project coverage”**: refers to the role of the evidence files that are used to extract the line
  and branch coverage for the project (all fuzz-targets combined). In FuzzBinder, we decided to use
  files in `gs://oss-fuzz-coverage/{projectName}/reports/{date}/linux/summary.json` for this
  purpose. These files contain fine-grained and aggregated coverage metrics.
- **“fuzzTarget coverage”**: refers to the role of the evidence files that are used to extract the
  line and branch coverage for a given fuzz-target. In FuzzBinder, we decided to use files in
  `gs://oss-fuzz-coverage/{projectName}/fuzzer_stats/{date}/{fuzz-target}.json` for this purpose.
  These files contain fine-grained and aggregated coverage metrics computed per fuzz-target.

Additional considerations:

- It is not necessary to add the fuzz-targets code in the evidence since it can be found on Github
  using the commit hash (**subject[*].digest.sha1**) and the fuzz-targets paths
  (**predicate.claimSpec.perTarget[*].path**).
- The URI of the evidence (used in the current implementation of FuzzBinder on GitHub) is based on
  the Google Cloud Storage paths mentioned above (`gs://…`). Therefore, the used fuzz-target and
  date (which are important for reproducing the claims) are included in the URI of the evidence.
- The logs that are used for the crash detection and the fuzzing effort computation are not included
  in the evidence since storing them is costly (hundreds of log files are generated per day).
  Therefore, the evidence for the crash detection and the fuzzing effort is the FuzzBinder code
  itself which is open-source. It is possible to check the dataflow in Fuzzbinder and make sure that
  all the log files of a given revision of the source code on a given date are analyzed to detect
  crashes and compute the fuzzing effort.

As mentioned above, the evidence files are currently stored in Google cloud Storage by OSS-Fuzz.
However, they are deleted after a given time period. Therefore, we need to store them permanently to
assure that they can be verified in the future. Since [Ent](https://github.com/google/ent) is a
permanent content-addressable store, it will be used for this purpose. To avoid tampering with the
evidence while copying it to [Ent](https://github.com/google/ent), FuzzBinder makes the copies of
the evidence that is used to compute the FuzzClaim specification from the original GCS bucket of
OSS-Fuzz or ClusterFuzz to [Ent](https://github.com/google/ent) and generates the claims. If the
original evidence is not available in the GCS bucket of OSS-Fuzz, FuzzBinder does not generate the
claims.

## Tool

In this section, design decisions regarding the code are mentioned.

- FuzzBinder makes use of fuzzing reports of [Cluster-fuzz](https://google.github.io/clusterfuzz/)
  and [OSS-Fuzz](https://github.com/google/oss-fuzz).
- FuzzBinder is a standalone command-line in Go as part of the Transparent-Release repository.
- FuzzBinder is open-source so that open source projects can use it, and integrate it into their
  build and release pipelines.
- FuzzBinder will be wrapped in a
  [GitHub workflow](https://docs.github.com/en/actions/using-workflows/about-workflows). It makes
  use of role-accounts to access the fuzzing data of each project (that uses FuzzBinder), and an
  [in-toto run action](https://github.com/marketplace/actions/in-toto-run) to sign the FuzzClaims
  and publish them in [Rekor](https://github.com/sigstore/rekor).

## Threat model

|                            | Untrusted                                                    | Trusted-but-verifiable                                              | Trusted                                        |
| :------------------------- | :----------------------------------------------------------- | :------------------------------------------------------------------ | :--------------------------------------------- |
| For FuzzBinder             | End users and The authors of the revision of the source code | Ent storage                                                         | OSS-fuzz, ClusterFuzz and Google Cloud Storage |
| For the user of FuzzBinder | End users                                                    | FuzzBinder, The authors of the revision of the code and Ent storage | OSS-fuzz, ClusterFuzz and Google Cloud Storage |

## How-To guide

A how-to use FuzzBinder guide is available [here](../cmd/fuzzbinder/README.md).

## Glossary

- **Fuzzing**: an automated testing technique that allows vulnerability detection by generating
  malformed inputs to trigger unwanted behaviors and find bugs in binaries.
- **Function coverage**: the percentage of functions which have been executed at least once. A
  function is considered to be executed if any of its instantiations are executed.
  [[source](https://clang.llvm.org/docs/SourceBasedCodeCoverage.html#interpreting-reports)]
- **Instantiation coverage**: the percentage of function instantiations which have been executed at
  least once. Template functions and static inline functions from headers are two kinds of functions
  which may have multiple instantiations.
  [[source](https://clang.llvm.org/docs/SourceBasedCodeCoverage.html#interpreting-reports)]
- **Line coverage**: the percentage of code lines which have been executed at least once. Only
  executable lines within function bodies are considered to be code lines.
  [[source](https://clang.llvm.org/docs/SourceBasedCodeCoverage.html#interpreting-reports)]
- **Region coverage**: the percentage of code regions which have been executed at least once. A code
  region may span multiple lines (e.g in a large function body with no control flow). However, it’s
  also possible for a single line to contain multiple code regions (e.g in “return x || y && z”).
  [[source](https://clang.llvm.org/docs/SourceBasedCodeCoverage.html#interpreting-reports)]
- **Branch coverage**: the percentage of “true” and “false” branches that have been taken at least
  once. Each branch is tied to specific conditions in the source code that may each evaluate to
  either “true” or “false”. These conditions may comprise larger boolean expressions linked by
  boolean logical operators. For example, “x = (y == 2) || (z < 10)” is a boolean expression that is
  composed of two individual conditions, each of which evaluates to either true or false, producing
  four total branch outcomes.
  [[source](https://clang.llvm.org/docs/SourceBasedCodeCoverage.html#interpreting-reports)]
- **Cyclomatic complexity**: a software metric used to indicate the complexity of a program. It is a
  quantitative measure of the number of linearly independent paths through a program's source code.
  [[source](https://en.wikipedia.org/wiki/Cyclomatic_complexity)]
