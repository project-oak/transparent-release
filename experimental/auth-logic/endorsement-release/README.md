This directory includes an executable binary for using authorization logic to decide if endorsement files can be produced.

It takes as input 

- One or more authorization logic files. These include a policy for releasing endorsement files written by the product team and zero or more other authorization logic files that might implement libraries.
- A provenance file for building the binary to be released.

It generates as output:
 - An authorization logic file that extends the input policies with the outputs of the provenance file wrapper and the unix time wrapper.

The build rule endorsement_release_oak_functions runs this binary with an example. The input files for this example are:
 - `experimental/auth-logic/endorsement-release/input_policy_examples/oak_endorsement_policy.auth_logic` -- the policy set by the Oak functions loader team for deciding when to release a binary
 - `experimental/auth-logic/endorsement-release/input_policy_examples/github_actions_policy.auth_logic`

The real output is generated in:
 - `bazel-bin/experimental/auth-logic/endorsement-release/oak_endorsement_release_output_policy.auth_logic`
and a copy of this is stored in:
 - `experimental/auth-logic/endorsement-release/output_policy_examples/oak_endorsement_release_output_policy.auth_logic`

This input policy example is somewhat more complicated than what is really needed -- it includes a "policy principal" that specifies when and how to trust outputs from particular builders like Github Actions and/or Google Cloud Build. This extra complexity was done deliberately to demonstrate:
 - the concept of policy principals which might have applications to other problems like: writing and agreeing to conform to a standard for certificate authorities, writing/conforming to identity policies (e.g., is this a cryptographer? is this a person with rust readability? is this an RFID reader with a particular specification?)
 - the value of authorization logic because it uses a delegation to a policy principal