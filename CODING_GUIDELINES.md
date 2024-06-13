# Coding Guidelines

This document is an extension to [CONTRIBUTING](./CONTRIBUTING.md) and provides more details about the coding guidelines and requirements.

## API & Design

* Code must be well structured:
    * packages must have a limited responsibility (different concerns can go to different packages),
    * types must be easy to compose,
    * think about maintainability and testability.
* "Depend upon abstractions, [not] concretions".
* Try to limit the number of methods you are exposing. It's easier to expose something later than to hide it.
* Take advantage of `internal` package concept.
* Follow agreed-upon design patterns and naming conventions.
* publicly-exposed functions are named logically, have forward-thinking arguments and return types.
* Avoid global variables and global configurators.
* Favor composable and extensible designs.
* Minimize code duplication.
* Limit third-party dependencies.

### Performance

* Avoid unnecessary operations or memory allocations.

### Security

* Pay proper attention to exploits involving:
    * gas usage
    * transaction verification and signatures
    * malleability
    * code must be always deterministic
* Thread safety. If some functionality is not thread-safe, or uses something that is not thread-safe, then clearly indicate the risk on each level.

### Documentation

When writing code that is complex or relies on another piece of the code, it is advised to create a diagram or a flowchart to explain the logic. This will help other developers to understand the code and will also help you to understand the logic better.

Lava uses [Mermaid.js](https://mermaid.js.org/), you can find the documentation on how to use it [here](https://mermaid.js.org/intro/).

## Acceptance tests

Start the design by defining Acceptance Tests. The purpose of Acceptance Testing is to
validate that the product being developed corresponds to the needs of the real users
and is ready for launch. Hence we often talk about **User Acceptance Test** (UAT).
It also gives a better understanding of the product and helps designing a right interface
and API.

UAT should be revisited at each stage of the product development:


### Why Acceptance Testing

* Automated acceptance tests catch serious problems that unit or component test suites could never catch.
* Automated acceptance tests deliver business value the users are expecting as they test user scenarios.
* Automated acceptance tests executed and passed on every build help improve the software delivery process.
* Testers, developers, and customers need to work closely to create suitable automated acceptance test suites.

### How to define Acceptance Test

The best way to define AT is by starting from the user stories and think about all positive and negative scenarios a user can perform.

Product Developers should collaborate with stakeholders to define AT. Functional experts and business users are both needed for defining AT.

A good pattern for defining AT is listing scenarios with [GIVEN-WHEN-THEN](https://martinfowler.com/bliki/GivenWhenThen.html) format where:

* **GIVEN**: A set of initial circumstances (e.g. bank balance)
* **WHEN**: Some event happens (e.g. customer attempts a transfer)
* **THEN**: The expected result as per the defined behavior of the system

In other words: we define a use case input, current state and the expected outcome. Example:

> Feature: User trades stocks.
> Scenario: User requests a sell before close of trading
>
>     Given I have 100 shares of MSFT stock
>        And I have 150 shares of APPL stock
>        And the time is before close of trading
>
>     When I ask to sell 20 shares of MSFT stock
>
>      Then I should have 80 shares of MSFT stock
>       And I should have 150 shares of APPL stock
>       And a sell order for 20 shares of MSFT stock should have been executed

*Reference: [writing acceptance tests](https://openclassrooms.com/en/courses/4544611-write-agile-documentation-user-stories-acceptance-tests/4810081-writing-acceptance-tests)*.

### How and where to add acceptance tests

Acceptance tests are written in the Markdown format, using the scenario template described above, and be part of the specification (`xx_test.md` file in *spec* directory). Example: [`eco-credits/spec/06.test.md`](https://github.com/regen-network/regen-ledger/blob/7297783577e6cd102c5093365b573163680f36a1/x/ecocredit/spec/06_tests.md).

Acceptance tests should be defined during the design phase or at an early stage of development. Moreover, they should be defined before writing a module architecture - it will clarify the purpose and usage of the software.
Automated tests should cover all acceptance tests scenarios.

## Automated Tests

Make sure your code is well tested:

* Provide unit tests for every unit of your code if possible. Unit tests are expected to comprise 70%-80% of your tests.
* Describe the test scenarios you are implementing for integration tests.
* Create integration tests for queries and msgs.
* Use both test cases and property / fuzzy testing.
* Do not decrease code test coverage. Explain in a PR if test coverage is decreased.

We expect tests to use `require` or `assert` rather than `t.Skip` or `t.Fail`,
unless there is a reason to do otherwise.
When testing a function under a variety of different inputs, we prefer to use
[table driven tests](https://github.com/golang/go/wiki/TableDrivenTests).
Table driven test error messages should follow the following format
`<desc>, tc #<index>, i #<index>`.
`<desc>` is an optional short description of what's failing, `tc` is the
index within the test case table that is failing, and `i` is when there
is a loop, exactly which iteration of the loop failed.
The idea is you should be able to see the
error message and figure out exactly what failed.
Here is an example check:

    ```go
        <some table>
        for tcIndex, tc := range cases {
        <some code>
            resp, err := doSomething()
            require.NoError(err)
            require.Equal(t, tc.expected, resp, "should correctly perform X")
    ```