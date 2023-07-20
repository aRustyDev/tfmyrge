* Create more tests
    - conflict w/ resolution set = override
    - conflict w/ resolution set = default
    - conflict w/ resolution set = takeFirstArg
    - conflict w/ resolution set = takeNewest
    - conflict w/ resolution set = takeOldest
    - conflict w/ resolution set = Merge
* Review current tests
    - failures: why are they returning nil?
    - failures: are they actually working?
* Should I keep the custom error handling & diff tracking?
    - if yes: create custom error structs in errors_test.go
* 