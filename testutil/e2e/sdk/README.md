## Lava SDK E2E

This repository hosts the code for the Lava SDK End-to-End (E2E) framework. All test cases should be situated within the /tests directory.

### Test Case Guidelines

Due to the error parsing strategy used in the Lava E2E framework, certain rules must be followed when authoring new test cases:

1. The Lava SDK E2E makes use of Standard Output (Stdout) and Standard Error (Stderr) for logging all operations in the 01_sdkTest.log file. Following the execution of all tests, this file will be parsed and analyzed for any errors. To enable the framework to identify an Error, the error message must be written as follows:

```
console.log(" ERR " + error.message) // Ensure there's a space before and after "ERR"
```

2. A variety of problems can occur during the initialization of the Lava SDK. To handle this, the test should be enclosed within a try/catch block:

```
(async () => {
    try {
        await main();
    } catch (error) {
        console.error(" ERR "+error.message);
        process.exit(1);
    }
})();
```

3. For importing the Lava SDK, we are utilizing a local binary:

```
const { LavaSDK } = require("../../../../ecosystem/lava-sdk/bin/src/sdk/sdk");
```

4. All environment variables are pre-stored in process.env and can be leveraged within the test:
   (current version pre-loads only privatekey)

```
privateKey: process.env.PRIVATE_KEY,
```