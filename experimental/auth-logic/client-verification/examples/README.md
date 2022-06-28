This directory contains an example output policy from the transparent release
verification binary that has the values from the wrappers filled in. 
This output example is in the file oak_verification_passing.auth_logic.

This example was created by running the following from the root directory of the
transparent release repository:
```
bazel build //experimental/auth-logic/client-verification:oak_verification_passing
cp bazel-bin/experimental/auth-logic/client-verification/oak_verification_passing.auth_logic \
    experimental/auth-logic/client-verification/oak_verification_passing.auth_logic
```