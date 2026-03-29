Run the test suite and analyze results.

Execute `task test` (or `task test:short` if the user passes "short" as argument: $ARGUMENTS).

If tests fail:
1. Read the failing test file to understand what's being tested
2. Read the source code under test
3. Identify and fix the root cause
4. Re-run only the failing test to confirm the fix

Do NOT modify tests to make them pass unless the test itself is wrong.
