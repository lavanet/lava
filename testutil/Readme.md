## Lava e2e tests


### Test Lava e2e Locally
```
go test ./testutil/e2e/ -v -timeout 240s

# (you can also run the tests from vscode)
```

#### Logs will be saved to 
```
lava/testutil/logs/processID.log
```

#### Github Actions file for e2e Test
```
lava/.github/workflows/e2e.yml
```

#### Create a new test
```
- create a new file called some_test.go                      # must end with _test.go
    - make a function: func TestSomething(t *testing.T)      # must start with Test

# checkout simple_test.go or lava_fullflow_test.go
```
